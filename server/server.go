package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	pb "distributed-key-value-store/proto"

	"google.golang.org/grpc"
)

type Log struct {
	key   string
	value string
	index int32
}

type StoreCommit struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Mixed struct {
	StrVal string
	IntVal int
}

type server struct {
	pb.UnimplementedKeyValueStoreServer
	mu                sync.Mutex
	store             map[string]string
	peers             []string
	selfIp            string
	leaderIp          string
	lastHeartbeatTime time.Time
	logs              []Log
	lastcommitedindex int32
	dataDir           string
}

func (s *server) GetLogIndex(ctx context.Context, req *pb.Empty) (*pb.LogIndexResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return &pb.LogIndexResponse{LogIndex: s.lastcommitedindex}, nil
}

func binarySearch(arr []Log, target int32) int {
	left, right := 0, len(arr)-1
	for left <= right {
		mid := left + (right-left)/2
		if arr[mid].index == target {
			return mid
		} else if arr[mid].index < target {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	return -1
}

func (s *server) ClearLogs(ctx context.Context, req *pb.ClearFromNum) (*pb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = s.logs[binarySearch(s.logs, req.FromNum):]
	return &pb.Empty{}, nil
}

func (s *server) SendMinLogIndex(ctx context.Context, req *pb.Empty) (*pb.MinLogIndexResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// From all servers get the minimum log index
	minIndex := s.lastcommitedindex
	for _, peer := range s.peers {
		if !isReachable(peer) {
			continue
		}
		conn, err := grpc.Dial(peer, grpc.WithInsecure())
		if err != nil {
			log.Printf("Warning: cannot connect to %s for heartbeat: %v", peer, err)
			continue
		}
		client := pb.NewKeyValueStoreClient(conn)
		response, err := client.GetLogIndex(context.Background(), &pb.Empty{})
		if err != nil {
			log.Printf("Warning: heartbeat failed to %s: %v", peer, err)
		}
		if response.LogIndex < minIndex {
			minIndex = response.LogIndex
		}
		conn.Close()
	}

	// Send the minimum log index to all servers
	for _, peer := range s.peers {
		if !isReachable(peer) {
			continue
		}
		conn, err := grpc.Dial(peer, grpc.WithInsecure())
		if err != nil {
			log.Printf("Warning: cannot connect to %s for heartbeat: %v", peer, err)
			continue
		}
		client := pb.NewKeyValueStoreClient(conn)
		_, err = client.ClearLogs(context.Background(), &pb.ClearFromNum{FromNum: minIndex})
		if err != nil {
			log.Printf("Warning: heartbeat failed to %s: %v", peer, err)
		}
		conn.Close()
	}
	return &pb.MinLogIndexResponse{MinLogIndex: minIndex}, nil
}

func NewServer(ip string, peers []string) *server {
	dataDir := "data_" + strings.Replace(ip, ":", "_", -1)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
	return &server{
		store:  make(map[string]string),
		peers:  peers,
		selfIp: ip,

		lastHeartbeatTime: time.Now(),
		logs:              []Log{},
		dataDir:           dataDir,
		lastcommitedindex: -1,
	}
}

func (s *server) CommitDatatoDisk() {
	if len(s.logs) == 0 {
		return
	}
	lastcommited := binarySearch(s.logs, s.lastcommitedindex)
	logtocommit := s.logs[lastcommited+1:]

	for _, log := range logtocommit {
		data, err := json.Marshal(StoreCommit{Key: log.key, Value: log.value})
		if err != nil {
			fmt.Printf("Warning: Could not serialize store data: %v", err)
			return
		}

		storeFile := filepath.Join(s.dataDir, "store.json")
		if err := ioutil.WriteFile(storeFile, data, 0644); err != nil {
			fmt.Printf("Warning: Could not write store file: %v", err)
		}

	}
	s.lastcommitedindex = s.logs[len(s.logs)-1].index
}

func (s *server) RegularLogCommit() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		s.CommitDatatoDisk()
		s.mu.Unlock()
	}
}

func (s *server) LogCommit(ctx context.Context, req *pb.LogCommitRequest) (*pb.LogCommitResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.LogIndex-1 != s.logs[len(s.logs)-1].index {
		s.syncWithLeader(s.logs[len(s.logs)-1].index+1, req.LogIndex)
		return &pb.LogCommitResponse{Success: false}, nil
	}

	s.store[req.Key] = req.Value
	s.logs = append(s.logs, Log{key: req.Key, value: req.Value, index: req.LogIndex})
	return &pb.LogCommitResponse{Success: true}, nil
}

func (s *server) SendLogCommitToPeers(key string, value string, logIndex int32) (success bool) {

	for _, peer := range s.peers {
		if peer == s.selfIp {
			continue
		}

		if !isReachable(peer) {
			continue
		}

		conn, err := grpc.Dial(peer, grpc.WithInsecure())
		if err != nil {
			log.Printf("Warning: cannot connect to %s to update leader: %v", peer, err)
			continue
		}
		client := pb.NewKeyValueStoreClient(conn)
		response, err := client.LogCommit(context.Background(), &pb.LogCommitRequest{Key: key, Value: value, LogIndex: logIndex})
		if !response.Success {
			log.Printf("Warning: Log commit failed on %s: %v", peer, err)
			return false
		}
	}
	return true
}

func isReachable(addr string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (s *server) electLeader() string {
	//create array with string and int
	available := []Mixed{}

	if isReachable(s.selfIp) {
		available = append(available, Mixed{StrVal: s.selfIp, IntVal: int(s.lastcommitedindex)})
	}

	// get last commited index from all servers
	for _, peer := range s.peers {
		if isReachable(peer) {
			conn, err := grpc.Dial(peer, grpc.WithInsecure())
			if err != nil {
				log.Printf("Warning: cannot connect to %s to get last committed index: %v", peer, err)
				continue
			}
			client := pb.NewKeyValueStoreClient(conn)
			resp, err := client.GetLogIndex(context.Background(), &pb.Empty{})
			if err != nil {
				log.Printf("Warning: cannot get last committed index from %s: %v", peer, err)
				conn.Close()
				continue
			}
			Ind := resp.LogIndex
			log.Printf("Last committed index from %s: %d", peer, Ind)
			Ip := peer
			conn.Close()
			available = append(available, Mixed{StrVal: Ip, IntVal: int(Ind)})
		}
	}

	if len(available) == 0 {
		log.Fatal("No available servers for leader election!")
	}

	sort.Slice(available, func(i, j int) bool {
		if available[i].IntVal == available[j].IntVal {
			return available[i].StrVal < available[j].StrVal
		}
		return available[i].IntVal < available[j].IntVal
	})
	newLeader := available[len(available)-1].StrVal
	return newLeader
}

func (s *server) notifyPeers(newLeader string) {
	for _, peer := range s.peers {
		if peer == s.selfIp {
			continue
		}

		if !isReachable(peer) {
			continue
		}
		conn, err := grpc.Dial(peer, grpc.WithInsecure())
		if err != nil {
			log.Printf("Warning: cannot connect to %s to update leader: %v", peer, err)
			continue
		}
		client := pb.NewKeyValueStoreClient(conn)
		_, err = client.UpdateLeader(context.Background(), &pb.UpdateLeaderRequest{LeaderIp: newLeader})
		if err != nil {
			log.Printf("Warning: cannot update leader on %s: %v", peer, err)
		}
		conn.Close()
	}
}

func (s *server) UpdateLeader(ctx context.Context, req *pb.UpdateLeaderRequest) (*pb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.leaderIp = req.LeaderIp
	log.Printf("Leader updated to %s", s.leaderIp)
	return &pb.Empty{}, nil
}

func (s *server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, found := s.store[req.Key]
	return &pb.GetResponse{Value: value, Found: found}, nil
}

func (s *server) Put(ctx context.Context, req *pb.PutRequest) (*pb.PutResponse, error) {
	if s.selfIp != s.leaderIp {
		conn, err := grpc.Dial(s.leaderIp, grpc.WithInsecure())
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		client := pb.NewKeyValueStoreClient(conn)
		return client.Put(ctx, req)
	}

	newIndex := s.lastcommitedindex + 1
	s.logs = append(s.logs, Log{key: req.Key, value: req.Value, index: newIndex})
	s.lastcommitedindex = newIndex

	s.mu.Lock()
	s.store = make(map[string]string)
	log.Printf("Put: %s -> %s", req.Key, req.Value)
	data, _ := json.Marshal(StoreCommit{Key: req.Key, Value: req.Value})
	storeFile := filepath.Join(s.dataDir, "store.json")
	// add data to store file

	if err := ioutil.WriteFile(storeFile, data, 0644); err != nil {
		fmt.Printf("Warning: Could not write store file: %v", err)
	}
	s.mu.Unlock()

	success := s.SendLogCommitToPeers(req.Key, req.Value, newIndex)
	if !success {
		log.Printf("Warning: Log commit failed on some peers")
		return &pb.PutResponse{Success: false}, nil
	}
	return &pb.PutResponse{Success: true}, nil
}

func (s *server) GetLogEntry(ctx context.Context, req *pb.GetLogEntryRequest) (*pb.GetLogEntryResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ind := req.Index
	ind = int32(binarySearch(s.logs, ind))
	if ind < 0 {
		return &pb.GetLogEntryResponse{Key: "Invalid", Value: "invalid", LogIndex: -1}, nil
	}
	return &pb.GetLogEntryResponse{Key: s.logs[ind].key, Value: s.logs[ind].value, LogIndex: req.Index}, nil
}

func (s *server) syncWithLeader(prevInd int32, currInd int32) {
	for i := prevInd; i <= currInd; i++ {
		log.Printf("Syncing with leader at index %d", i)
		Ip := s.leaderIp
		//get log entry from leader
		conn, err := grpc.Dial(Ip, grpc.WithInsecure())
		if err != nil {
			log.Printf("Warning: unable to connect to %s: %v", Ip, err)
			continue
		}
		client := pb.NewKeyValueStoreClient(conn)
		resp, err := client.GetLogEntry(context.Background(), &pb.GetLogEntryRequest{Index: i})
		if err != nil {
			log.Printf("Warning: replication error to %s: %v", Ip, err)
		}
		if resp.LogIndex != -1 {
			s.mu.Lock()
			s.logs = append(s.logs, Log{
				key:   resp.Key,
				value: resp.Value,
				index: resp.LogIndex,
			})
			s.mu.Unlock()
		}

		conn.Close()
	}
}

func (s *server) Replicate(ctx context.Context, req *pb.ReplicateRequest) (*pb.Empty, error) {
	s.mu.Lock()
	s.store[req.Key] = req.Value
	s.mu.Unlock()
	return &pb.Empty{}, nil
}

func (s *server) Heartbeat(ctx context.Context, req *pb.Empty) (*pb.Empty, error) {
	s.mu.Lock()
	s.lastHeartbeatTime = time.Now()
	s.mu.Unlock()
	return &pb.Empty{}, nil
}

func (s *server) SendandReceiveHeartbeat() {

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		currentLeader := s.leaderIp
		selfIp := s.selfIp
		lastHB := s.lastHeartbeatTime
		s.mu.Unlock()

		if selfIp == currentLeader {
			for _, peer := range s.peers {
				if peer == selfIp {
					continue
				}
				if !isReachable(peer) {
					continue
				}
				conn, err := grpc.Dial(peer, grpc.WithInsecure())
				if err != nil {
					log.Printf("Warning: cannot connect to %s for heartbeat: %v", peer, err)
					continue
				}
				client := pb.NewKeyValueStoreClient(conn)
				_, err = client.Heartbeat(context.Background(), &pb.Empty{})
				if err != nil {
					log.Printf("Warning: heartbeat failed to %s: %v", peer, err)
				}
				conn.Close()
			}
		} else {

			elapsed := time.Since(lastHB)
			log.Printf("Time elapsed since last heartbeat from leader (%s): %v", currentLeader, elapsed)

			if elapsed > 8*time.Second {
				newLeader := s.electLeader()
				s.mu.Lock()
				s.leaderIp = newLeader
				s.lastHeartbeatTime = time.Now()
				s.mu.Unlock()
				log.Printf("Leader elected: %s", newLeader)

				if newLeader == selfIp {
					s.notifyPeers(newLeader)
				}
			}
		}
	}
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: server <self-ip> <peer1> <peer2> ...")
	}

	selfIp := os.Args[1]
	peers := os.Args[2:]

	// Start listening first.
	lis, err := net.Listen("tcp", selfIp)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	srv := NewServer(selfIp, peers)
	pb.RegisterKeyValueStoreServer(grpcServer, srv)

	go func() {
		fmt.Printf("Server listening at %s\n", selfIp)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	time.Sleep(1 * time.Second)

	newLeader := srv.electLeader()
	srv.mu.Lock()
	srv.leaderIp = newLeader
	srv.mu.Unlock()
	log.Printf("Leader elected: %s", newLeader)

	if srv.selfIp == newLeader {
		srv.notifyPeers(newLeader)
	}

	go srv.SendandReceiveHeartbeat()

	go srv.RegularLogCommit()

	select {}
}

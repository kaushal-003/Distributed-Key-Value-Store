package main

import "C"
import (
	"context"
	pb "distributed-key-value-store/proto"
	"fmt"
	"regexp"
	"sync"
	"time"
	"unsafe"

	"google.golang.org/grpc"
)

var (
	servers    []string
	currServer string
	conn       *grpc.ClientConn
	mutex      sync.Mutex
)

func ValidateKey(key string) bool {
	if len(key) > 128 {
		return false
	}
	for _, ch := range key {
		if ch < 32 || ch > 126 || ch == '[' || ch == ']' {
			return false
		}
	}
	return true
}

func ValidateValue(value string) bool {
	if len(value) > 2048 {
		return false
	}

	validValuePattern := regexp.MustCompile(`^[a-zA-Z0-9\s.,_-]+$`)
	return validValuePattern.MatchString(value)
}

func kv_init(serverList **C.char) C.int {
	mutex.Lock()
	defer mutex.Unlock()

	servers = []string{}
	ptr := serverList

	for *ptr != nil {
		servers = append(servers, C.GoString(*ptr))
		ptr = (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + unsafe.Sizeof(*ptr)))
	}

	for _, server := range servers {
		var err error
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		conn, err = grpc.DialContext(ctx, server, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			continue
		} else {
			fmt.Println("Connected to server:", server)
			currServer = server
			return 0
		}
	}

	fmt.Println("No available servers to connect to.")
	return -1
}

func kv_shutdown() C.int {
	mutex.Lock()
	defer mutex.Unlock()

	if conn != nil {
		//close grpc conn
		conn.Close()
		conn = nil
	}
	servers = nil
	return 0
}

func kv_get(key *C.char, value *C.char) C.int {
	var err2 error
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err2 = grpc.DialContext(ctx, currServer, grpc.WithInsecure(), grpc.WithBlock())
	if err2 != nil {
		for _, server := range servers {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			conn, err2 = grpc.DialContext(ctx, server, grpc.WithInsecure(), grpc.WithBlock())
			if err2 != nil {
				continue
			} else {
				fmt.Println("Connected to server:", server)
				currServer = server
				break
			}
		}
	}

	if err2 != nil {
		return -1
	}

	fmt.Sprintf("GET %s\n", C.GoString(key))

	if !ValidateKey(C.GoString(key)) {
		fmt.Println("Invalid key: Keys must be printable ASCII (without '[' or ']') and ≤ 128 bytes")
		return -1
	}

	client := pb.NewKeyValueStoreClient(conn)
	resp, err := client.Get(context.Background(), &pb.GetRequest{Key: C.GoString(key)})

	if err != nil {
		return -1
	}
	val := resp.Value
	if !resp.Found {
		return 1
	}

	cResponse := C.CString(val)
	defer C.free(unsafe.Pointer(cResponse))
	C.strcpy(value, cResponse)
	return 0
}

func kv_put(key *C.char, value *C.char, old_value *C.char) C.int {
	mutex.Lock()
	defer mutex.Unlock()

	var err2 error
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err2 = grpc.DialContext(ctx, currServer, grpc.WithInsecure(), grpc.WithBlock())
	if err2 != nil {
		for _, server := range servers {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			conn, err2 = grpc.DialContext(ctx, server, grpc.WithInsecure(), grpc.WithBlock())
			if err2 != nil {
				continue
			} else {
				fmt.Println("Connected to server:", server)
				currServer = server
				break
			}
		}
	}

	if err2 != nil {
		return -1
	}

	old_status := kv_get(key, old_value)
	if old_status == -1 {
		return -1
	}

	if !ValidateKey(C.GoString(key)) {
		fmt.Println("Invalid key: Keys must be printable ASCII (without '[' or ']') and ≤ 128 bytes")
		return -1
	}

	if !ValidateValue(C.GoString(value)) {
		fmt.Println("Invalid value: Values must be printable ASCII and ≤ 2048 bytes")
		return -1
	}

	client := pb.NewKeyValueStoreClient(conn)
	_, err := client.Put(context.Background(), &pb.PutRequest{Key: C.GoString(key), Value: C.GoString(value)})

	fmt.Println("Value put successfully")
	if err != nil {
		return -1
	}

	if old_status == 0 {
		return 0
	}
	return 1

}

func main() {

}

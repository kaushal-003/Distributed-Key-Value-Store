package main

import (
	"context"
	"fmt"
	"log"
	"os"

	pb "distributed-key-value-store/proto"

	"google.golang.org/grpc"
)

func main() {
	if len(os.Args) < 4 {
		log.Fatalf("Usage: client <server-ip> <operation> <key> [value]")
	}

	serverIp := os.Args[1]
	operation := os.Args[2]
	key := os.Args[3]

	conn, err := grpc.Dial(serverIp, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewKeyValueStoreClient(conn)

	switch operation {
	case "get":
		resp, err := client.Get(context.Background(), &pb.GetRequest{Key: key})
		if err != nil {
			log.Fatalf("Error getting value: %v", err)
		}
		if resp.Found {
			fmt.Printf("Value: %s\n", resp.Value)
		} else {
			fmt.Println("Key not found")
		}
	case "put":
		if len(os.Args) < 5 {
			log.Fatalf("Usage: client <server-ip> put <key> <value>")
		}
		value := os.Args[4]
		_, err := client.Put(context.Background(), &pb.PutRequest{Key: key, Value: value})
		if err != nil {
			log.Fatalf("Error setting value: %v", err)
		}
		fmt.Println("Value set successfully")
	default:
		log.Fatalf("Unknown operation: %s", operation)
	}
}

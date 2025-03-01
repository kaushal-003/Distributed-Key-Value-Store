package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"

	pb "distributed-key-value-store/proto"

	"google.golang.org/grpc"
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

func main() {
	if len(os.Args) < 4 {
		log.Fatalf("Usage: client <server-ip> <operation> <key> [value]")
	}

	serverIp := os.Args[1]
	operation := os.Args[2]
	key := os.Args[3]

	if !ValidateKey(key) {
		log.Fatalf("Invalid key: Keys must be printable ASCII (without '[' or ']') and ≤ 128 bytes")
	}

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
		if !ValidateValue(value) {
			log.Fatalf("Invalid value: Values must be printable ASCII (without special characters) and ≤ 2048 bytes")
		}
		_, err := client.Put(context.Background(), &pb.PutRequest{Key: key, Value: value})
		if err != nil {
			log.Fatalf("Error setting value: %v", err)
		}
		fmt.Println("Value set successfully")
	default:
		log.Fatalf("Unknown operation: %s", operation)
	}
}

package main

/*
#include <stdlib.h>
#include <string.h>
*/
import "C"
import (
	"context"
	pb "distributed-key-value-store/proto"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"google.golang.org/grpc"
)

// Global variables for managing state
var (
	servers    []string
	currServer string
	conn       *grpc.ClientConn
	mutex      sync.Mutex
)

//export kv_init
func kv_init(serverList **C.char) C.int {
	mutex.Lock()
	defer mutex.Unlock()

	servers = []string{}
	ptr := serverList

	// Convert C string array to Go slice
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

//export kv_shutdown
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

//export kv_get
func kv_get(key *C.char, value *C.char) C.int {
	// conn, err := grpc.Dial(currServer, grpc.WithInsecure())
	// if err != nil {
	// 	for _, server := range servers {
	// 		var err error
	// 		conn, err = grpc.Dial(server, grpc.WithInsecure())
	// 		if err != nil {
	// 			continue
	// 		} else {
	// 			fmt.Println("Connected to server:", server)
	// 			currServer = server
	// 			break
	// 		}
	// 	}
	// }

	// if err != nil {
	// 	return -1
	// }
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

	//Send GET request to server
	fmt.Sprintf("GET %s\n", C.GoString(key))
	client := pb.NewKeyValueStoreClient(conn)

	resp, err := client.Get(context.Background(), &pb.GetRequest{Key: C.GoString(key)})

	if err != nil {
		return -1
	}
	val := resp.Value
	if !resp.Found {
		return 1
	}

	// Convert Go string to C string and copy to the provided buffer
	cResponse := C.CString(val)
	defer C.free(unsafe.Pointer(cResponse))
	C.strcpy(value, cResponse)
	return 0
}

//export kv_put
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

	// if conn == nil {
	// 	return -1
	// }

	old_status := kv_get(key, old_value)
	if old_status == -1 {
		return -1
	}

	// Send PUT request
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

} // Required for cgo shared libraries

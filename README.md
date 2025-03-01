# Distributed-Key-Value-Store
This repository contains code for distributed key-value store as a part of course CS-616 Distributed Systems.

## Server Setup

1. Start mongodb on each host
```
sudo systemctl start mongod
```

2. Run the server file from `server` directory on each server

```
cd server
go run server.go <self ip> <peer1 ip:port> <peer2 ip:port> ... <peerN ip:port>
```

- for eg.
```
cd server
go run server.go 127.0.0.1:5000 127.0.0.1:5001 127.0.0.1:5002
```

3. To run test cases, run following command. We have given many testfiles such as `perfTest.c`, `recoverytest.c`,`correctness.c` and `recoveryresult.c`. You can also create a custom test file using the exported funtions `kv_init()`, `kv_get()`, `kv_put()` and `kv_shutdown`.

```
cd tests
gcc -o <testfile> <testfile.c> ./my-libkv.so -Wl,-rpath=.
```

## Performance Results:
1. While running multiple servers on single machine
  ![image](https://github.com/user-attachments/assets/772a61b8-e8d8-4e7f-aed8-88d1d066c769)

3. While running multiple servers on different machines
![image](https://github.com/user-attachments/assets/f72491bd-6a84-4c03-9d41-1a74ffb3f458)

Authors:
1. Kaushal Kothiya - 21110107
2. Anish Karnik - 21110098

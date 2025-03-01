// test.c
#include <stdio.h>
#include "my-libkv.h"  // Header file generated when building my-libkv.so

int main(void) {
    // Define a null-terminated array of server strings.
    char *servers[] = {"127.0.0.1:5000", "127.0.0.1:5001", "127.0.0.1:5002", NULL};

    // Initialize the key/value client.
    int result = kv_init(servers);
    if (result == 0) {
        printf("kv_init succeeded.\n");
    } else {
        printf("kv_init failed with error code: %d\n", result);
        return 1;
    }

    // Prepare a buffer to receive the value from kv_get.
    char value[1024];

    // Issue a GET request for the key "testkey".
    result = kv_get("testkey1", value);
    if (result == 0) {
        // Key found, print the returned value.
        printf("kv_get succeeded: value for 'testkey' is '%s'\n", value);
    } else if (result == 1) {
        // Key not found.
        printf("kv_get: key 'testkey' not found\n");
    } else {
        // An error occurred.
        printf("kv_get failed with error code: %d\n", result);
    }

    // Issue a PUT request for the key "testkey".

    strcpy(value, "newvalue1");

    char oldvalue[1024];
    result = kv_put("testkey1", value, oldvalue);
    if (result == 0) {
        // Key found, print the returned value.
        printf("kv_put succeeded: value for 'testkey' is '%s'\n and old value is '%s'\n", value, oldvalue);
    } else if (result == 1) {
        // Key not found.
        printf("kv_put: key 'testkey' not found\n");
    } else {
        // An error occurred.
        printf("kv_put failed with error code: %d\n", result);
    }

    // Issue a GET request for the key "testkey".
    result = kv_get("testkey1", value);
    if (result == 0) {
        // Key found, print the returned value.
        printf("kv_get succeeded: value for 'testkey' is '%s'\n", value);
    } else if (result == 1) {
        // Key not found.
        printf("kv_get: key 'testkey' not found\n");
    } else {
        // An error occurred.
        printf("kv_get failed with error code: %d\n", result);
    }

    strcpy(value, "newvalue2");
    result = kv_put("testkey1", value, oldvalue);
    if (result == 0) {
        // Key found, print the returned value.
        printf("kv_put succeeded: value for 'testkey' is '%s'\n and old value is '%s'\n", value, oldvalue);
    } else if (result == 1) {
        // Key not found.
        printf("kv_put: key 'testkey' not found\n");
    } else {
        // An error occurred.
        printf("kv_put failed with error code: %d\n", result);
    }

    // Shutdown the key/value client.
    //
    kv_shutdown();
    return 0;
}

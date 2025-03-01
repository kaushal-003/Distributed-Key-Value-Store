#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "my-libkv.h"  


int main(void) {
    // Define a null-terminated array of server strings.
    char *servers[] = {"127.0.0.1:5000", "127.0.0.1:5001", "127.0.0.1:5002", NULL};

    printf("Initializing the key-value store\n");
    int result = kv_init(servers);
    if (result == 0) {
        printf("kv_init succeeded.\n");
    } else {
        printf("kv_init failed with error code: %d\n", result);
        return 1;
    }

    char value[1024];
    
    printf("Putting the key-value pair\n");
    strcpy(value, "Yesssir");
    char oldvalue[1024];
    result = kv_put("Recovery", value, oldvalue);

    printf("Run the recoveryresult.c to find out whether after shutting the leader down, the system is still able to serve the requests\n");
    printf("Make sure you dont put the leader IP in the servers list of recoveryresult.c \n");
    return 0;
}
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "my-libkv.h"

void run_test_case(char *desc, char* done) {
    printf("Expected Result: %s, Got: %s\n", desc, done);
    printf("%s\n\n", (strcmp(desc, done) == 0) ? "✅ Passed" : "❌ Failed");
}

int main(void) {
    char *servers[] = {"127.0.0.1:5002", NULL};

    printf("Initializing the key-value store\n");
    int result = kv_init(servers);
    if (result == 0) {
        printf("kv_init succeeded.\n");
    } else {
        printf("kv_init failed with error code: %d\n", result);
        return 1;
    }

    char value[1024];
    int get_status = kv_get("Recovery", value);
    run_test_case("Yesssir", value);
    printf("Stored Value: %s\n\n", value);

    return 0;
}

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "my-libkv.h"

char *servers[] = {"127.0.0.1:5000", "127.0.0.1:5001", "127.0.0.1:5002", NULL};

void run_test_case(int test_num, const char *desc, int expected, int result) {
    printf("Test Case %d: %s\n", test_num, desc);
    printf("Expected Result: %d, Got: %d\n", expected, result);
    printf("%s\n\n", (expected == result) ? "✅ Passed" : "❌ Failed");
}

int main() {
    int init_status = kv_init(servers);
    run_test_case(1, "Initialize KV Store", 0, init_status);
    if (init_status != 0) {
        printf("❌ KV store initialization failed. Exiting...\n");
        return -1;
    }

    char value[2048];
    char old_value[2048];

    // Test 2: Valid PUT request
    int put_status = kv_put("validKey", "TestValue", old_value);
    run_test_case(2, "PUT valid key-value pair", 1, put_status);

    // Test 3: GET existing key
    int get_status = kv_get("validKey", value);
    run_test_case(3, "GET existing key", 0, get_status);
    printf("Stored Value: %s\n\n", value);

    // Test 4: GET non-existing key
    int get_nonexistent_status = kv_get("nonExistentKey", value);
    run_test_case(4, "GET non-existent key", 1, get_nonexistent_status);

    // Test 5: Overwrite existing key
    int put_overwrite_status = kv_put("validKey", "NewValue", old_value);
    run_test_case(5, "Overwrite existing key", 0, put_overwrite_status);
    printf("Old Value: %s\n\n", old_value);

    // Test 6: Validate GET after overwrite
    int get_updated_status = kv_get("validKey", value);
    run_test_case(6, "GET after overwrite", 0, get_updated_status);
    printf("Updated Value: %s\n\n", value);

    // Test 7: Invalid Key (contains '[')
    int invalid_key_status = kv_put("invalid[key]", "SomeValue", old_value);
    run_test_case(7, "PUT with invalid key (contains '[')", -1, invalid_key_status);

    // Test 8: Invalid Value (contains special characters)
    int invalid_value_status = kv_put("validKey2", "Invalid@Value!", old_value);
    run_test_case(8, "PUT with invalid value (contains special characters)", -1, invalid_value_status);

    // Test 9: Shutdown and restart
    int shutdown_status = kv_shutdown();
    run_test_case(9, "Shutdown KV Store", 0, shutdown_status);

    int reinit_status = kv_init(servers);
    run_test_case(10, "Reinitialize KV Store", 0, reinit_status);

    // Test 11: GET after shutdown (check persistence)
    int get_persistence_status = kv_get("validKey", value);
    run_test_case(11, "GET after restart (check persistence)", 0, get_persistence_status);
    printf("Persistent Value: %s\n\n", value);

    // Final Shutdown
    kv_shutdown();

    return 0;
}

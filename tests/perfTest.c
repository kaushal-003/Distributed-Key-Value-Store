#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include "my-libkv.h"

#define NUM_KEYS 100
#define NUM_OPS 10000

long long get_ns() {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (long long) ts.tv_sec * 1000000000LL + ts.tv_nsec;
}

int main(void) {
    int result;
    char key[256];
    char value[1024];
    char oldvalue[1024];

    char *servers[] = {"10.7.41.69:5000", "10.7.28.104:5000", NULL};
    result = kv_init(servers);
    if (result != 0) {
        printf("kv_init failed with error code: %d\n", result);
        return 1;
    }
    printf("kv_init succeeded.\n");

    printf("Inserting %d keys into the database...\n", NUM_KEYS);
    for (int i = 0; i < NUM_KEYS; i++) {
        snprintf(key, sizeof(key), "key%d", i);
        snprintf(value, sizeof(value), "value%d", i);
        result = kv_put(key, value, oldvalue);
        if (result < 0) {
            printf("kv_put failed for key %s with error code: %d\n", key, result);
        }
    }
    printf("Insertion complete.\n");

    long long start, end, duration;
    double throughput;
    double avg_latency;  // in microseconds
    int errors = 0;

    printf("\nPerformance Test: Uniformly Random Distribution\n");
    srand((unsigned int) time(NULL));
    start = get_ns();
    errors = 0;
    for (int i = 0; i < NUM_OPS; i++) {
        int idx = rand() % NUM_KEYS;
        snprintf(key, sizeof(key), "key%d", idx);
        result = kv_get(key, value);
        if (result < 0) {
            errors++;
        }
    }
    end = get_ns();
    duration = end - start;
    throughput = NUM_OPS / (duration / 1e9);
    avg_latency = (duration / 1000.0) / NUM_OPS;
    printf("Uniform Random: %d ops in %lld ns, throughput = %.2f ops/sec, avg latency = %.2f µs, errors = %d\n",
           NUM_OPS, duration, throughput, avg_latency, errors);

    printf("\nPerformance Test: Hot/Cold Distribution (10%% hot, 90%% of requests on hot keys)\n");
    start = get_ns();
    errors = 0;
    for (int i = 0; i < NUM_OPS; i++) {
        int r = rand() % 100;
        if (r < 90) {
            // 90%: choose a random hot key among key0 to key9.
            int idx = rand() % 10;
            snprintf(key, sizeof(key), "key%d", idx);
        } else {
            // 10%: choose a random cold key among key10 to key99.
            int idx = 10 + (rand() % (NUM_KEYS - 10));
            snprintf(key, sizeof(key), "key%d", idx);
        }
        result = kv_get(key, value);
        if (result < 0) {
            errors++;
        }
    }
    end = get_ns();
    duration = end - start;
    throughput = NUM_OPS / (duration / 1e9);
    avg_latency = (duration / 1000.0) / NUM_OPS;
    printf("Hot/Cold: %d ops in %lld ns, throughput = %.2f ops/sec, avg latency = %.2f µs, errors = %d\n",
           NUM_OPS, duration, throughput, avg_latency, errors);

    kv_shutdown();
    return 0;
}

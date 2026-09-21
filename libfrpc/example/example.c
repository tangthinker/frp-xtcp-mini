/*
 * Example: compile against the shared library after `make libfrpc`.
 *
 *   cc -o /tmp/libfrpc-example libfrpc/example/example.c -I libfrpc bin/libfrpc.dylib -Wl,-rpath,$PWD/bin
 *   /tmp/libfrpc-example '{"serverAddr":"127.0.0.1","serverPort":7000,"auth":{"method":"token","token":"12345678"}}'
 */

#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

#include "frpc.h"

static void on_event(int handle, const char *json, void *userdata) {
	(void)userdata;
	fprintf(stderr, "event handle=%d %s\n", handle, json);
}

int main(int argc, char **argv) {
	const char *config =
		"{"
		"\"serverAddr\":\"127.0.0.1\","
		"\"serverPort\":7000,"
		"\"auth\":{\"method\":\"token\",\"token\":\"12345678\"},"
		"\"visitors\":[{"
		"\"name\":\"ssh-visitor\","
		"\"type\":\"xtcp\","
		"\"serverName\":\"ssh\","
		"\"secretKey\":\"abcdefg\","
		"\"bindAddr\":\"127.0.0.1\","
		"\"bindPort\":0,"
		"\"keepTunnelOpen\":true"
		"}]"
		"}";
	if (argc > 1) {
		config = argv[1];
	}

	printf("libfrpc %s\n", frpc_version());

	int handle = 0;
	int rc = frpc_start(config, &handle);
	if (rc != FRPC_OK) {
		fprintf(stderr, "start failed: %s\n", frpc_strerror(rc));
		return 1;
	}

	frpc_set_event_callback(handle, on_event, NULL);
	rc = frpc_wait_connected(handle, 15000);
	if (rc != FRPC_OK) {
		fprintf(stderr, "wait connected: %s\n", frpc_strerror(rc));
		frpc_stop(handle);
		return 1;
	}

	char *status = NULL;
	if (frpc_status(handle, &status) == FRPC_OK) {
		printf("%s\n", status);
		frpc_free(status);
	}

	sleep(1);
	frpc_stop(handle);
	return 0;
}

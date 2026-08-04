package main

import "testing"

func TestWorkServerLoopbackAddress(t *testing.T) {
	getenv := func(name string) string {
		if name == "WORK_BIND_HOST" {
			return "127.0.0.1"
		}
		return ""
	}
	if got := workServerListenAddress(getenv, "8080"); got != "127.0.0.1:8080" {
		t.Fatalf("loopback listen address = %q", got)
	}
	if got := workServerListenAddress(func(string) string { return "" }, "8080"); got != ":8080" {
		t.Fatalf("default listen address = %q", got)
	}
}

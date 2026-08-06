package main

import (
	"net"
	"strings"
	"testing"
	"time"
)

func TestWorkServerLoopbackAddress(t *testing.T) {
	getenv := func(name string) string {
		if name == "WORK_BIND_HOST" {
			return "127.0.0.1"
		}
		return ""
	}
	got, err := workServerListenAddress(getenv, "8080")
	if err != nil {
		t.Fatal(err)
	}
	if got != "127.0.0.1:8080" {
		t.Fatalf("loopback listen address = %q", got)
	}
	got, err = workServerListenAddress(func(string) string { return "" }, "8080")
	if err != nil {
		t.Fatal(err)
	}
	if got != ":8080" {
		t.Fatalf("default listen address = %q", got)
	}
}

func TestWorkServerBindHostNormalization(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		want    string
		wantErr string
	}{
		{name: "DNS", host: "localhost", want: "localhost:8080"},
		{name: "raw IPv4", host: "127.0.0.1", want: "127.0.0.1:8080"},
		{name: "raw IPv6", host: "::1", want: "[::1]:8080"},
		{name: "bracketed IPv6", host: "[::1]", want: "[::1]:8080"},
		{name: "bracketed IPv6 zone", host: "[fe80::1%lo]", want: "[fe80::1%lo]:8080"},
		{name: "missing closing bracket", host: "[::1", wantErr: "unmatched"},
		{name: "missing opening bracket", host: "::1]", wantErr: "unmatched"},
		{name: "nested bracket", host: "[[::1]]", wantErr: "malformed"},
		{name: "bracketed DNS", host: "[localhost]", wantErr: "valid IPv6"},
		{name: "bracketed IPv4", host: "[127.0.0.1]", wantErr: "valid IPv6"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := workServerListenAddress(func(name string) string {
				if name == "WORK_BIND_HOST" {
					return test.host
				}
				return ""
			}, "8080")
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, test.wantErr)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("address = %q, error = %v, want %q", got, err, test.want)
			}
		})
	}
}

func TestWorkServerLoopbackReachability(t *testing.T) {
	address, err := workServerListenAddress(func(name string) string {
		if name == "WORK_BIND_HOST" {
			return "127.0.0.1"
		}
		return ""
	}, "0")
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatalf("listen on %q: %v", address, err)
	}
	defer listener.Close()

	deadline := time.Now().Add(2 * time.Second)
	if err := listener.(*net.TCPListener).SetDeadline(deadline); err != nil {
		t.Fatal(err)
	}
	accepted := make(chan error, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr == nil {
			_ = connection.SetDeadline(deadline)
			acceptErr = connection.Close()
		}
		accepted <- acceptErr
	}()

	connection, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("dial loopback listener: %v", err)
	}
	_ = connection.SetDeadline(deadline)
	if err := connection.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-accepted; err != nil {
		t.Fatalf("accept loopback connection: %v", err)
	}
}

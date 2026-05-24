package doctor

import (
	"context"
	"net"
	"testing"
)

func TestCheckTCPListenerAcceptsRunningListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp: %v", err)
	}
	defer listener.Close()

	err = checkTCPListener(context.Background(), listener.Addr().String())
	if err != nil {
		t.Fatalf("checkTCPListener() error = %v", err)
	}
}

func TestCheckUDPListenerAcceptsBoundPort(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen udp: %v", err)
	}
	defer conn.Close()

	status, err := checkUDPListener(conn.LocalAddr().String())
	if err != nil {
		t.Fatalf("checkUDPListener() error = %v", err)
	}
	if status != StatusSkipped {
		t.Fatalf("udp status = %q, want %q", status, StatusSkipped)
	}
}

func TestCheckUDPListenerRejectsInvalidAddress(t *testing.T) {
	_, err := checkUDPListener("not a hostport")
	if err == nil {
		t.Fatal("checkUDPListener() error = nil, want parse error")
	}
}

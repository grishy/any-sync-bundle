package doctor

import (
	"context"
	"fmt"
	"net"
	"time"
)

func checkTCPListener(ctx context.Context, listenAddr string) error {
	host, port, err := net.SplitHostPort(listenAddr)
	if err != nil {
		return fmt.Errorf("parse tcp listen address: %w", err)
	}
	dialHost := host
	if host == "0.0.0.0" {
		dialHost = "127.0.0.1"
	}
	if host == "::" {
		dialHost = "::1"
	}

	dialCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	var dialer net.Dialer
	conn, err := dialer.DialContext(dialCtx, "tcp", net.JoinHostPort(dialHost, port))
	if err != nil {
		return err
	}
	return conn.Close()
}

func checkUDPListener(listenAddr string) (HealthStatus, error) {
	if _, _, err := net.SplitHostPort(listenAddr); err != nil {
		return StatusProblem, fmt.Errorf("parse udp listen address: %w", err)
	}
	return StatusSkipped, nil
}

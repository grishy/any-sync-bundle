package doctor

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
)

func unixHTTPClient(socketPath string) *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _ string, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", socketPath)
		},
		DisableCompression: true,
	}
	return &http.Client{Transport: transport}
}

func RunClient(ctx context.Context, socketPath string, out io.Writer) error {
	fmt.Fprintln(out, "Connecting to running bundle")
	fmt.Fprintf(out, "  socket: %s\n", socketPath)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://doctor/doctor/run", nil)
	if err != nil {
		return fmt.Errorf("create doctor request: %w", err)
	}

	client := unixHTTPClient(socketPath)
	defer client.CloseIdleConnections()

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("connect to doctor socket %q: %w", socketPath, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("doctor request failed: %s: %s", response.Status, string(body))
	}
	fmt.Fprintln(out, "  status: connected")
	fmt.Fprintln(out)

	_, copyErr := io.Copy(out, response.Body)
	if copyErr != nil {
		return fmt.Errorf("read doctor stream: %w", copyErr)
	}

	return nil
}

package adminui

// Config represents admin UI configuration.
type Config struct {
	// ListenAddr is the address to listen on (e.g., "0.0.0.0:8888")
	ListenAddr string `yaml:"listenAddr"`
}

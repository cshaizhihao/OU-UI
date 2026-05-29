package config

import (
	"fmt"
	"os"
	"strings"
)

type ServerConfig struct {
	Host           string
	Port           string
	DatabasePath   string
	AdminUser      string
	AdminPassword  string
	JWTSecret      string
	AgentJoinToken string
	SecurePath     string
	TLSCertFile    string
	TLSKeyFile     string
}

func LoadServer() ServerConfig {
	securePath := getenv("OUUI_SECURE_PATH", "/ou-ui")
	if !strings.HasPrefix(securePath, "/") {
		securePath = "/" + securePath
	}

	return ServerConfig{
		Host:           getenv("OUUI_HOST", "0.0.0.0"),
		Port:           getenv("OUUI_PORT", "8080"),
		DatabasePath:   getenv("OUUI_DB", "ou-ui.db"),
		AdminUser:      getenv("OUUI_ADMIN_USER", "admin"),
		AdminPassword:  getenv("OUUI_ADMIN_PASSWORD", "change-me-now"),
		JWTSecret:      getenv("OUUI_JWT_SECRET", "dev-only-change-me"),
		AgentJoinToken: getenv("OUUI_AGENT_JOIN_TOKEN", "dev-agent-token"),
		SecurePath:     securePath,
		TLSCertFile:    os.Getenv("OUUI_TLS_CERT_FILE"),
		TLSKeyFile:     os.Getenv("OUUI_TLS_KEY_FILE"),
	}
}

func (c ServerConfig) ListenAddr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func (c ServerConfig) TLSEnabled() bool {
	return c.TLSCertFile != "" && c.TLSKeyFile != ""
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

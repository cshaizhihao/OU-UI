package hysteria2

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type Config struct {
	Listen           string            `json:"listen"`
	TLS              TLSConfig         `json:"tls"`
	Auth             AuthConfig        `json:"auth"`
	Bandwidth        *BandwidthConfig  `json:"bandwidth,omitempty"`
	Masquerade       *MasqueradeConfig `json:"masquerade,omitempty"`
	UserTrafficLimit *TrafficLimit     `json:"userTrafficLimit,omitempty"`
}

type TLSConfig struct {
	CertPath string `json:"certPath"`
	KeyPath  string `json:"keyPath"`
}

type AuthConfig struct {
	Password string `json:"password"`
}

type BandwidthConfig struct {
	Up                    string `json:"up,omitempty"`
	Down                  string `json:"down,omitempty"`
	IgnoreClientBandwidth bool   `json:"ignoreClientBandwidth,omitempty"`
}

type MasqueradeConfig struct {
	Type        string            `json:"type"`
	File        *MasqueradeFile   `json:"file,omitempty"`
	Proxy       *MasqueradeProxy  `json:"proxy,omitempty"`
	String      *MasqueradeString `json:"string,omitempty"`
	ListenHTTP  string            `json:"listenHTTP,omitempty"`
	ListenHTTPS string            `json:"listenHTTPS,omitempty"`
	ForceHTTPS  bool              `json:"forceHTTPS,omitempty"`
}

type MasqueradeFile struct {
	Dir string `json:"dir"`
}

type MasqueradeProxy struct {
	URL         string `json:"url"`
	RewriteHost bool   `json:"rewriteHost,omitempty"`
	Insecure    bool   `json:"insecure,omitempty"`
	XForwarded  bool   `json:"xForwarded,omitempty"`
}

type MasqueradeString struct {
	Content    string            `json:"content"`
	Headers    map[string]string `json:"headers,omitempty"`
	StatusCode int               `json:"statusCode,omitempty"`
}

type TrafficLimit struct {
	Enabled    bool   `json:"enabled,omitempty"`
	Up         string `json:"up,omitempty"`
	Down       string `json:"down,omitempty"`
	TotalBytes uint64 `json:"totalBytes,omitempty"`
}

type specView struct {
	Listen           string            `json:"listen"`
	Port             int               `json:"port"`
	TLS              TLSConfig         `json:"tls"`
	TLSCertPath      string            `json:"tlsCertPath"`
	TLSKeyPath       string            `json:"tlsKeyPath"`
	Auth             AuthConfig        `json:"auth"`
	AuthPassword     string            `json:"authPassword"`
	Password         string            `json:"password"`
	Bandwidth        *BandwidthConfig  `json:"bandwidth"`
	Masquerade       *MasqueradeConfig `json:"masquerade"`
	UserTrafficLimit *TrafficLimit     `json:"userTrafficLimit"`
	TrafficLimit     *TrafficLimit     `json:"trafficLimit"`
}

func ConfigFromNode(listen string, port int, settings map[string]any) (Config, error) {
	view, err := decodeSettings(settings)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Listen: strings.TrimSpace(firstString(view.Listen, listen)),
		TLS: TLSConfig{
			CertPath: strings.TrimSpace(firstString(view.TLS.CertPath, view.TLSCertPath)),
			KeyPath:  strings.TrimSpace(firstString(view.TLS.KeyPath, view.TLSKeyPath)),
		},
		Auth: AuthConfig{
			Password: firstString(view.Auth.Password, view.AuthPassword, view.Password),
		},
		Bandwidth:        view.Bandwidth,
		Masquerade:       view.Masquerade,
		UserTrafficLimit: firstTrafficLimit(view.UserTrafficLimit, view.TrafficLimit),
	}

	if cfg.Listen == "" {
		if view.Port != 0 {
			port = view.Port
		}
		if port != 0 {
			cfg.Listen = ":" + strconv.Itoa(port)
		}
	}

	return cfg, nil
}

func ValidateConfig(cfg Config) error {
	var problems []string

	if cfg.Listen == "" {
		problems = append(problems, "listen port is required")
	} else if err := validateListenPort(cfg.Listen); err != nil {
		problems = append(problems, err.Error())
	}

	if cfg.TLS.CertPath == "" {
		problems = append(problems, "tls cert path is required")
	}
	if cfg.TLS.KeyPath == "" {
		problems = append(problems, "tls key path is required")
	}
	if cfg.Auth.Password == "" {
		problems = append(problems, "auth password is required")
	}

	if cfg.Masquerade != nil {
		if err := validateMasquerade(*cfg.Masquerade); err != nil {
			problems = append(problems, err.Error())
		}
	}

	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}

	return nil
}

func decodeSettings(settings map[string]any) (specView, error) {
	if len(settings) == 0 {
		return specView{}, nil
	}

	raw, err := json.Marshal(settings)
	if err != nil {
		return specView{}, fmt.Errorf("encode hysteria2 settings: %w", err)
	}

	var view specView
	if err := json.Unmarshal(raw, &view); err != nil {
		return specView{}, fmt.Errorf("decode hysteria2 settings: %w", err)
	}

	return view, nil
}

func validateListenPort(listen string) error {
	port, err := portFromListen(listen)
	if err != nil {
		return err
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("listen port must be between 1 and 65535: %d", port)
	}
	return nil
}

func portFromListen(listen string) (int, error) {
	listen = strings.TrimSpace(listen)
	if listen == "" {
		return 0, errors.New("listen port is required")
	}
	if strings.HasPrefix(listen, "realm://") {
		return 0, errors.New("realm listen URIs are not supported by the v0.3.0 provider")
	}
	if strings.Contains(listen, "-") {
		return 0, errors.New("listen port ranges are not supported by the v0.3.0 provider")
	}
	if n, err := strconv.Atoi(listen); err == nil {
		return n, nil
	}

	_, portText, err := net.SplitHostPort(listen)
	if err != nil {
		if strings.HasPrefix(listen, ":") {
			portText = strings.TrimPrefix(listen, ":")
		} else {
			return 0, fmt.Errorf("listen must include a valid port: %q", listen)
		}
	}

	port, err := strconv.Atoi(portText)
	if err != nil {
		return 0, fmt.Errorf("listen must include a numeric port: %q", listen)
	}

	return port, nil
}

func validateMasquerade(masq MasqueradeConfig) error {
	switch masq.Type {
	case "":
		return errors.New("masquerade type is required when masquerade is configured")
	case "file":
		if masq.File == nil || strings.TrimSpace(masq.File.Dir) == "" {
			return errors.New("masquerade file.dir is required")
		}
	case "proxy":
		if masq.Proxy == nil || strings.TrimSpace(masq.Proxy.URL) == "" {
			return errors.New("masquerade proxy.url is required")
		}
		if parsed, err := url.Parse(masq.Proxy.URL); err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("masquerade proxy.url must be a valid absolute URL: %q", masq.Proxy.URL)
		}
	case "string":
		if masq.String == nil {
			return errors.New("masquerade string content is required")
		}
	default:
		return fmt.Errorf("unsupported masquerade type %q", masq.Type)
	}

	return nil
}

func firstString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstTrafficLimit(values ...*TrafficLimit) *TrafficLimit {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

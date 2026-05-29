package hysteria2

import (
	"sort"
	"strconv"
	"strings"
)

func RenderYAML(cfg Config) []byte {
	var b strings.Builder

	writeKV(&b, 0, "listen", cfg.Listen)

	writeLine(&b, 0, "tls:")
	writeKV(&b, 1, "cert", cfg.TLS.CertPath)
	writeKV(&b, 1, "key", cfg.TLS.KeyPath)

	writeLine(&b, 0, "auth:")
	writeKV(&b, 1, "type", "password")
	writeKV(&b, 1, "password", cfg.Auth.Password)

	if cfg.Bandwidth != nil && (cfg.Bandwidth.Up != "" || cfg.Bandwidth.Down != "") {
		writeLine(&b, 0, "bandwidth:")
		if cfg.Bandwidth.Up != "" {
			writeKV(&b, 1, "up", cfg.Bandwidth.Up)
		}
		if cfg.Bandwidth.Down != "" {
			writeKV(&b, 1, "down", cfg.Bandwidth.Down)
		}
	}
	if cfg.Bandwidth != nil && cfg.Bandwidth.IgnoreClientBandwidth {
		writeBool(&b, 0, "ignoreClientBandwidth", true)
	}

	if cfg.Masquerade != nil {
		renderMasquerade(&b, *cfg.Masquerade)
	}

	return []byte(b.String())
}

func renderMasquerade(b *strings.Builder, masq MasqueradeConfig) {
	writeLine(b, 0, "masquerade:")
	writeKV(b, 1, "type", masq.Type)

	switch masq.Type {
	case "file":
		writeLine(b, 1, "file:")
		writeKV(b, 2, "dir", masq.File.Dir)
	case "proxy":
		writeLine(b, 1, "proxy:")
		writeKV(b, 2, "url", masq.Proxy.URL)
		writeBool(b, 2, "rewriteHost", masq.Proxy.RewriteHost)
		writeBool(b, 2, "insecure", masq.Proxy.Insecure)
		writeBool(b, 2, "xForwarded", masq.Proxy.XForwarded)
	case "string":
		writeLine(b, 1, "string:")
		writeKV(b, 2, "content", masq.String.Content)
		if len(masq.String.Headers) > 0 {
			writeLine(b, 2, "headers:")
			keys := make([]string, 0, len(masq.String.Headers))
			for key := range masq.String.Headers {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				writeKV(b, 3, key, masq.String.Headers[key])
			}
		}
		if masq.String.StatusCode != 0 {
			writeInt(b, 2, "statusCode", masq.String.StatusCode)
		}
	}

	if masq.ListenHTTP != "" {
		writeKV(b, 1, "listenHTTP", masq.ListenHTTP)
	}
	if masq.ListenHTTPS != "" {
		writeKV(b, 1, "listenHTTPS", masq.ListenHTTPS)
	}
	if masq.ForceHTTPS {
		writeBool(b, 1, "forceHTTPS", true)
	}
}

func writeLine(b *strings.Builder, level int, text string) {
	b.WriteString(strings.Repeat("  ", level))
	b.WriteString(text)
	b.WriteByte('\n')
}

func writeKV(b *strings.Builder, level int, key string, value string) {
	writeLine(b, level, key+": "+quoteYAML(value))
}

func writeBool(b *strings.Builder, level int, key string, value bool) {
	writeLine(b, level, key+": "+strconv.FormatBool(value))
}

func writeInt(b *strings.Builder, level int, key string, value int) {
	writeLine(b, level, key+": "+strconv.Itoa(value))
}

func quoteYAML(value string) string {
	return strconv.Quote(value)
}

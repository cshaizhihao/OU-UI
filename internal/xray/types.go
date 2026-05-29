package xray

import "github.com/cshaizhihao/OU-UI/internal/provider"

const (
	ProtocolVLESS       = "vless"
	ProtocolVMess       = "vmess"
	ProtocolTrojan      = "trojan"
	ProtocolShadowsocks = "shadowsocks"
)

type Config struct {
	ID       string       `json:"id,omitempty"`
	Protocol string       `json:"protocol"`
	Listen   string       `json:"listen,omitempty"`
	Port     int          `json:"port"`
	Remark   string       `json:"remark,omitempty"`
	VLESS    *VLESSConfig `json:"vless,omitempty"`
	VMess    *VMessConfig `json:"vmess,omitempty"`
	Trojan   *TrojanConfig `json:"trojan,omitempty"`
	SS       *SSConfig     `json:"shadowsocks,omitempty"`
	Reality  *Reality     `json:"reality,omitempty"`
}

type VLESSConfig struct {
	UUID       string `json:"uuid"`
	Flow       string `json:"flow,omitempty"`
	Encryption string `json:"encryption,omitempty"`
}

type VMessConfig struct {
	UUID     string `json:"uuid"`
	AlterID  int    `json:"alterId,omitempty"`
	Security string `json:"security,omitempty"`
}

type TrojanConfig struct {
	Password string `json:"password"`
}

type SSConfig struct {
	Method   string `json:"method"`
	Password string `json:"password"`
	Network  string `json:"network,omitempty"`
}

type Reality struct {
	Enabled     bool     `json:"enabled"`
	Dest        string   `json:"dest"`
	ServerNames []string `json:"serverNames,omitempty"`
	PrivateKey  string   `json:"privateKey"`
	PublicKey   string   `json:"publicKey,omitempty"`
	ShortIDs    []string `json:"shortIds,omitempty"`
	SpiderX     string   `json:"spiderX,omitempty"`
}

type Provider struct{}

var _ provider.Provider = Provider{}

func NewProvider() Provider {
	return Provider{}
}

func (Provider) Name() provider.Runtime {
	return provider.RuntimeXray
}

package models

import (
	"time"

	"gorm.io/datatypes"
)

type Agent struct {
	ID            string         `gorm:"primaryKey" json:"id"`
	Name          string         `json:"name"`
	Version       string         `json:"version"`
	Status        string         `json:"status"`
	Hostname      string         `json:"hostname"`
	OS            string         `json:"os"`
	Arch          string         `json:"arch"`
	Kernel        string         `json:"kernel"`
	CPUModel      string         `json:"cpuModel"`
	CPUCount      int            `json:"cpuCount"`
	MemoryTotal   uint64         `json:"memoryTotal"`
	SwapTotal     uint64         `json:"swapTotal"`
	PublicIP      string         `json:"publicIp"`
	Capabilities  datatypes.JSON `json:"capabilities"`
	LastMetrics   datatypes.JSON `json:"lastMetrics"`
	LastSeenAt    *time.Time     `json:"lastSeenAt"`
	TrafficLimit  uint64         `json:"trafficLimit"`
	AgentTokenSHA string         `json:"-"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

type Task struct {
	ID        string         `gorm:"primaryKey" json:"id"`
	AgentID   string         `json:"agentId"`
	Type      string         `json:"type"`
	Status    string         `json:"status"`
	Payload   datatypes.JSON `json:"payload"`
	Result    datatypes.JSON `json:"result"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"createdAt"`
}

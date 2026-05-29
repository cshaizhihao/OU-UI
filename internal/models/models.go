package models

import (
	"time"

	"gorm.io/datatypes"
)

const (
	TaskTypeNoop          = "noop"
	TaskTypeRuntimeStatus = "runtime.status"
	TaskTypeNodeDeploy    = "node.deploy"
)

const (
	TaskStatusQueued    = "queued"
	TaskStatusRunning   = "running"
	TaskStatusSucceeded = "succeeded"
	TaskStatusFailed    = "failed"
	TaskStatusCanceled  = "canceled"
)

func IsSupportedTaskType(taskType string) bool {
	switch taskType {
	case TaskTypeNoop, TaskTypeRuntimeStatus, TaskTypeNodeDeploy:
		return true
	default:
		return false
	}
}

func IsAgentTaskUpdateStatus(status string) bool {
	switch status {
	case TaskStatusRunning, TaskStatusSucceeded, TaskStatusFailed:
		return true
	default:
		return false
	}
}

func IsTerminalTaskStatus(status string) bool {
	switch status {
	case TaskStatusSucceeded, TaskStatusFailed, TaskStatusCanceled:
		return true
	default:
		return false
	}
}

const (
	AgentStatusOnline   = "online"
	AgentStatusDegraded = "degraded"
	AgentStatusOffline  = "offline"
)

const (
	AgentAuthActive  = "active"
	AgentAuthRevoked = "revoked"
)

type Agent struct {
	ID            string         `gorm:"primaryKey" json:"id"`
	InstallID     string         `gorm:"index" json:"installId"`
	Name          string         `json:"name"`
	Version       string         `json:"version"`
	Status        string         `json:"status"`
	AuthStatus    string         `json:"authStatus"`
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
	LastError     string         `json:"lastError"`
	AgentTokenSHA string         `json:"-"`
	QueueCount    int            `gorm:"-" json:"queue"`
	Stale         bool           `gorm:"-" json:"stale"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

type Task struct {
	ID             string         `gorm:"primaryKey" json:"id"`
	AgentID        string         `gorm:"index:idx_tasks_agent_status_created" json:"agentId"`
	Type           string         `json:"type"`
	Status         string         `gorm:"index:idx_tasks_agent_status_created" json:"status"`
	Payload        datatypes.JSON `json:"payload"`
	Result         datatypes.JSON `json:"result"`
	Logs           string         `json:"logs"`
	Attempts       int            `json:"attempts"`
	MaxAttempts    int            `json:"maxAttempts"`
	LastError      string         `json:"lastError"`
	StartedAt      *time.Time     `json:"startedAt"`
	LeaseExpiresAt *time.Time  `json:"leaseExpiresAt"`
	FinishedAt     *time.Time     `json:"finishedAt"`
	CreatedAt      time.Time      `gorm:"index:idx_tasks_agent_status_created" json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type Node struct {
	ID             string         `gorm:"primaryKey" json:"id"`
	AgentID        string         `json:"agentId"`
	Name           string         `json:"name"`
	Runtime        string         `json:"runtime"`
	Protocol       string         `json:"protocol"`
	Status         string         `json:"status"`
	Spec           datatypes.JSON `json:"spec"`
	LastTaskID     string         `json:"lastTaskId"`
	RuntimeVersion string         `json:"runtimeVersion"`
	ServiceName    string         `json:"serviceName"`
	ServiceStatus  string         `json:"serviceStatus"`
	ConfigPath     string         `json:"configPath"`
	ConfigDir      string         `json:"configDir"`
	UnitPath       string         `json:"unitPath"`
	ServiceMode    string         `json:"serviceMode"`
	ManagedByOUUI  bool           `json:"managedByOuui"`
	LastError      string         `json:"lastError"`
	LastDeployedAt *time.Time     `json:"lastDeployedAt"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"createdAt"`
}

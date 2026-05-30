package models

import (
	"time"

	"gorm.io/datatypes"
)

const (
	TaskTypeNoop          = "noop"
	TaskTypeRuntimeStatus = "runtime.status"
	TaskTypeNodeDeploy    = "node.deploy"
	TaskTypeHostOptimize  = "host.optimize"
	TaskTypeRoutingApply  = "routing.apply"
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
	case TaskTypeNoop, TaskTypeRuntimeStatus, TaskTypeNodeDeploy, TaskTypeHostOptimize, TaskTypeRoutingApply:
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
	LeaseExpiresAt *time.Time     `json:"leaseExpiresAt"`
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

type NodeTrafficSample struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	NodeID      string    `gorm:"index:idx_node_traffic_node_collected" json:"nodeId"`
	AgentID     string    `gorm:"index" json:"agentId"`
	RxBytes     uint64    `json:"rxBytes"`
	TxBytes     uint64    `json:"txBytes"`
	RxRateBps   uint64    `json:"rxRateBps"`
	TxRateBps   uint64    `json:"txRateBps"`
	Connections int       `json:"connections"`
	CollectedAt time.Time `gorm:"index:idx_node_traffic_node_collected" json:"collectedAt"`
	CreatedAt   time.Time `json:"createdAt"`
}

type RoutingRule struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	Name        string    `json:"name"`
	Enabled     bool      `json:"enabled"`
	Priority    int       `gorm:"index" json:"priority"`
	RuleType    string    `json:"ruleType"`
	Match       string    `json:"match"`
	Protocol    string    `json:"protocol"`
	Action      string    `json:"action"`
	TargetTag   string    `json:"targetTag"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type LoadBalancerGroup struct {
	ID                  string         `gorm:"primaryKey" json:"id"`
	Name                string         `json:"name"`
	EntryTag            string         `json:"entryTag"`
	Strategy            string         `json:"strategy"`
	Members             datatypes.JSON `json:"members"`
	Status              string         `json:"status"`
	LastDecision        datatypes.JSON `json:"lastDecision"`
	HealthCheckInterval int            `json:"healthCheckInterval"`
	CreatedAt           time.Time      `json:"createdAt"`
	UpdatedAt           time.Time      `json:"updatedAt"`
}

type WebhookEndpoint struct {
	ID         string         `gorm:"primaryKey" json:"id"`
	Name       string         `json:"name"`
	Kind       string         `json:"kind"`
	URL        string         `json:"url"`
	Secret     string         `json:"-"`
	ChatID     string         `json:"chatId"`
	Enabled    bool           `json:"enabled"`
	EventTypes datatypes.JSON `json:"eventTypes"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

type AlertEvent struct {
	ID          string         `gorm:"primaryKey" json:"id"`
	Severity    string         `json:"severity"`
	SourceType  string         `json:"sourceType"`
	SourceID    string         `json:"sourceId"`
	EventType   string         `json:"eventType"`
	Message     string         `json:"message"`
	Payload     datatypes.JSON `json:"payload"`
	Delivered   bool           `json:"delivered"`
	DeliveredAt *time.Time     `json:"deliveredAt"`
	LastError   string         `json:"lastError"`
	CreatedAt   time.Time      `json:"createdAt"`
}

type ExternalSubscription struct {
	ID            string     `gorm:"primaryKey" json:"id"`
	Name          string     `json:"name"`
	URL           string     `json:"url"`
	Format        string     `json:"format"`
	Enabled       bool       `json:"enabled"`
	LastFetchedAt *time.Time `json:"lastFetchedAt"`
	LastError     string     `json:"lastError"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type ExternalNode struct {
	ID             string         `gorm:"primaryKey" json:"id"`
	SubscriptionID string         `gorm:"index" json:"subscriptionId"`
	Name           string         `json:"name"`
	Protocol       string         `json:"protocol"`
	Address        string         `json:"address"`
	Port           int            `json:"port"`
	Region         string         `json:"region"`
	Source         string         `json:"source"`
	Config         datatypes.JSON `json:"config"`
	Enabled        bool           `json:"enabled"`
	LatencyMs      int            `json:"latencyMs"`
	LossPercent    float64        `json:"lossPercent"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type ClashProfile struct {
	ID            string         `gorm:"primaryKey" json:"id"`
	Name          string         `json:"name"`
	RuleProviders datatypes.JSON `json:"ruleProviders"`
	ProxyGroups   datatypes.JSON `json:"proxyGroups"`
	RoutingRules  datatypes.JSON `json:"routingRules"`
	GeneratedYAML string         `json:"generatedYaml"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

type Tenant struct {
	ID                  string         `gorm:"primaryKey" json:"id"`
	Name                string         `json:"name"`
	Status              string         `json:"status"`
	Role                string         `json:"role"`
	NodeAccess          datatypes.JSON `json:"nodeAccess"`
	MonthlyTrafficQuota uint64         `json:"monthlyTrafficQuota"`
	PerNodeTrafficQuota uint64         `json:"perNodeTrafficQuota"`
	MaxConnections      int            `json:"maxConnections"`
	CreatedAt           time.Time      `json:"createdAt"`
	UpdatedAt           time.Time      `json:"updatedAt"`
}

type PanelUser struct {
	ID                  string         `gorm:"primaryKey" json:"id"`
	TenantID            string         `gorm:"index" json:"tenantId"`
	Username            string         `gorm:"uniqueIndex" json:"username"`
	PasswordSHA         string         `json:"-"`
	Role                string         `json:"role"`
	Status              string         `json:"status"`
	NodeAccess          datatypes.JSON `json:"nodeAccess"`
	MonthlyTrafficQuota uint64         `json:"monthlyTrafficQuota"`
	PerNodeTrafficQuota uint64         `json:"perNodeTrafficQuota"`
	MaxConnections      int            `json:"maxConnections"`
	CreatedAt           time.Time      `json:"createdAt"`
	UpdatedAt           time.Time      `json:"updatedAt"`
}

type APIKey struct {
	ID         string         `gorm:"primaryKey" json:"id"`
	TenantID   string         `gorm:"index" json:"tenantId"`
	Name       string         `json:"name"`
	KeyHash    string         `json:"-"`
	Scopes     datatypes.JSON `json:"scopes"`
	Status     string         `json:"status"`
	LastUsedAt *time.Time     `json:"lastUsedAt"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

type CopilotIncident struct {
	ID        string         `gorm:"primaryKey" json:"id"`
	Question  string         `json:"question"`
	Context   datatypes.JSON `json:"context"`
	Answer    string         `json:"answer"`
	Model     string         `json:"model"`
	Status    string         `json:"status"`
	CreatedAt time.Time      `json:"createdAt"`
}

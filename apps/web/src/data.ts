export type AgentStatus = "online" | "degraded" | "offline";

export type Runtime = "Xray" | "Hysteria2";

export type RuntimeRef =
  | Runtime
  | string
  | {
      name?: string;
      type?: Runtime | string;
      capabilities?: string[];
      runtimeCapabilities?: string[];
      runtime_capabilities?: string[];
    };

export type Protocol = "VLESS Reality" | "VMess" | "Trojan" | "Shadowsocks" | "Hysteria2";

export type ControlTaskStatus = "pending" | "running" | "success" | "failed";
export type LegacyTaskStatus = "queued" | "done";

export type AgentControlTask = {
  id?: string;
  status?: ControlTaskStatus | LegacyTaskStatus | string;
  state?: ControlTaskStatus | LegacyTaskStatus | string;
  failureReason?: string;
  failure_reason?: string;
  error?: string;
  errorMessage?: string;
  error_message?: string;
  retryCount?: number;
  retry_count?: number;
  retries?: number;
  attempts?: number;
};

export type Agent = {
  id: string;
  name: string;
  region: string;
  status: AgentStatus;
  runtime: RuntimeRef;
  ip: string;
  cpu: number;
  memory: number;
  uplinkMbps: number;
  downlinkMbps: number;
  usedTrafficGb: number;
  quotaTrafficGb: number;
  queue: number;
  updatedAt: string;
  registrationStatus?: string;
  registration_status?: string;
  registerStatus?: string;
  registrationState?: string;
  registration_state?: string;
  registered?: boolean;
  isRegistered?: boolean;
  is_registered?: boolean;
  authStatus?: string;
  auth_status?: string;
  authenticationStatus?: string;
  authentication_status?: string;
  authenticated?: boolean;
  isAuthenticated?: boolean;
  is_authenticated?: boolean;
  lastHeartbeat?: string;
  lastHeartbeatAt?: string;
  last_heartbeat?: string;
  last_heartbeat_at?: string;
  heartbeatAt?: string;
  heartbeat_at?: string;
  runtimeCapabilities?: string[];
  runtime_capabilities?: string[];
  capabilities?: string[];
  controlTask?: AgentControlTask;
  control_task?: AgentControlTask;
  currentTask?: AgentControlTask;
  current_task?: AgentControlTask;
  task?: AgentControlTask;
  taskStatus?: ControlTaskStatus | LegacyTaskStatus | string;
  task_status?: ControlTaskStatus | LegacyTaskStatus | string;
  failureReason?: string;
  failure_reason?: string;
  retryCount?: number;
  retry_count?: number;
  retries?: number;
};

export type DeployTask = {
  id: string;
  agentId: string;
  agentName: string;
  runtime: RuntimeRef;
  protocol: Protocol;
  action: string;
  status?: ControlTaskStatus | LegacyTaskStatus | string;
  state?: ControlTaskStatus | LegacyTaskStatus | string;
  progress?: number;
  eta?: string;
  failureReason?: string;
  failure_reason?: string;
  error?: string;
  errorMessage?: string;
  error_message?: string;
  retryCount?: number;
  retry_count?: number;
  retries?: number;
  attempts?: number;
};

export const agents: Agent[] = [
  {
    id: "ou-hkg-01",
    name: "Hong Kong Edge 01",
    region: "HK / HGC",
    status: "online",
    runtime: "Xray",
    ip: "10.18.4.21",
    cpu: 34,
    memory: 58,
    uplinkMbps: 182,
    downlinkMbps: 416,
    usedTrafficGb: 684,
    quotaTrafficGb: 1200,
    queue: 3,
    updatedAt: "18s ago",
    registrationStatus: "registered",
    authStatus: "authenticated",
    lastHeartbeat: "18s ago",
    runtimeCapabilities: ["hot reload", "reality keys", "config dry-run"],
    controlTask: {
      status: "running",
      retryCount: 0
    }
  },
  {
    id: "ou-sin-02",
    name: "Singapore Transit 02",
    region: "SG / Equinix",
    status: "online",
    runtime: "Hysteria2",
    ip: "10.21.9.44",
    cpu: 51,
    memory: 64,
    uplinkMbps: 236,
    downlinkMbps: 528,
    usedTrafficGb: 921,
    quotaTrafficGb: 1600,
    queue: 5,
    updatedAt: "42s ago",
    registrationStatus: "registered",
    authStatus: "authenticated",
    lastHeartbeat: "42s ago",
    runtimeCapabilities: ["port hopping", "udp relay", "bandwidth policy"],
    controlTask: {
      status: "pending",
      retryCount: 1
    }
  },
  {
    id: "ou-tyo-03",
    name: "Tokyo Relay 03",
    region: "JP / SoftBank",
    status: "degraded",
    runtime: "Xray",
    ip: "10.30.7.18",
    cpu: 76,
    memory: 72,
    uplinkMbps: 91,
    downlinkMbps: 204,
    usedTrafficGb: 1036,
    quotaTrafficGb: 1200,
    queue: 9,
    updatedAt: "2m ago",
    registrationStatus: "registered",
    authStatus: "expired",
    lastHeartbeat: "2m ago",
    runtimeCapabilities: ["certificate sync", "tls fingerprint", "inbound patch"],
    controlTask: {
      status: "failed",
      failureReason: "Certificate chain precheck failed",
      retryCount: 2
    }
  },
  {
    id: "ou-lax-04",
    name: "Los Angeles Exit 04",
    region: "US / LAX",
    status: "offline",
    runtime: "Hysteria2",
    ip: "10.42.2.11",
    cpu: 0,
    memory: 0,
    uplinkMbps: 0,
    downlinkMbps: 0,
    usedTrafficGb: 438,
    quotaTrafficGb: 1000,
    queue: 0,
    updatedAt: "maintenance",
    registrationStatus: "pending",
    authStatus: "unauthenticated",
    lastHeartbeat: "maintenance window",
    runtimeCapabilities: [],
    controlTask: {
      status: "success",
      retryCount: 0
    }
  }
];

export const protocolOptions: Protocol[] = [
  "VLESS Reality",
  "VMess",
  "Trojan",
  "Shadowsocks",
  "Hysteria2"
];

export const runtimeOptions: Runtime[] = ["Xray", "Hysteria2"];

export const taskQueue: DeployTask[] = [
  {
    id: "job-23061",
    agentId: "ou-hkg-01",
    agentName: "Hong Kong Edge 01",
    runtime: "Xray",
    protocol: "VLESS Reality",
    action: "Generate inbound and Reality shortId",
    state: "running",
    progress: 72,
    eta: "1m 20s",
    retryCount: 0
  },
  {
    id: "job-23062",
    agentId: "ou-sin-02",
    agentName: "Singapore Transit 02",
    runtime: "Hysteria2",
    protocol: "Hysteria2",
    action: "Refresh port-hopping policy",
    state: "pending",
    progress: 18,
    eta: "3m 45s",
    retryCount: 1
  },
  {
    id: "job-23058",
    agentId: "ou-tyo-03",
    agentName: "Tokyo Relay 03",
    runtime: "Xray",
    protocol: "Trojan",
    action: "Roll out certificate chain",
    state: "success",
    progress: 100,
    eta: "done",
    retryCount: 0
  },
  {
    id: "job-23057",
    agentId: "ou-tyo-03",
    agentName: "Tokyo Relay 03",
    runtime: "Xray",
    protocol: "Trojan",
    action: "Apply certificate chain",
    state: "failed",
    progress: 100,
    eta: "retry scheduled",
    failureReason: "Certificate chain precheck failed",
    retryCount: 2
  }
];

export const nodeHealthRows = [
  { name: "Inbound handshake success", value: "99.32%", detail: "Reality / TLS fingerprint check" },
  { name: "Queue wait median", value: "18s", detail: "Last 15 minutes" },
  { name: "Dispatchable agents", value: "3 / 4", detail: "1 agent in maintenance" }
];

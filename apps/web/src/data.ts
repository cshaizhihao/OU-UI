export type AgentStatus = "online" | "degraded" | "offline";

export type Runtime = "Xray" | "Hysteria2";

export type RuntimeApplyStage =
  | "render"
  | "install"
  | "apply"
  | "reload"
  | "health"
  | "rollback";
export type RuntimeServiceStatus =
  | "running"
  | "reloading"
  | "degraded"
  | "stopped"
  | "maintenance"
  | string;
export type RuntimeServiceMode = "managed" | "external" | string;

export type RuntimeApplyPhase = {
  stage?: RuntimeApplyStage | string;
  name?: RuntimeApplyStage | string;
  phase?: RuntimeApplyStage | string;
  status?: ControlTaskStatus | LegacyTaskStatus | string;
  state?: ControlTaskStatus | LegacyTaskStatus | string;
};

export type RuntimeApplySnapshot = {
  stage?: RuntimeApplyStage | string;
  applyStage?: RuntimeApplyStage | string;
  apply_stage?: RuntimeApplyStage | string;
  currentStage?: RuntimeApplyStage | string;
  current_stage?: RuntimeApplyStage | string;
  failureStage?: RuntimeApplyStage | string;
  failure_stage?: RuntimeApplyStage | string;
  failedStage?: RuntimeApplyStage | string;
  failed_stage?: RuntimeApplyStage | string;
  runtimeVersion?: string;
  runtime_version?: string;
  version?: string;
  serviceStatus?: RuntimeServiceStatus;
  service_status?: RuntimeServiceStatus;
  service?: RuntimeServiceStatus;
  serviceMode?: RuntimeServiceMode;
  service_mode?: RuntimeServiceMode;
  mode?: RuntimeServiceMode;
  runtimeManaged?: boolean | string | number;
  runtime_managed?: boolean | string | number;
  managedByOuUi?: boolean | string | number;
  managedByOuui?: boolean | string | number;
  managed_by_ou_ui?: boolean | string | number;
  unitPath?: string;
  unit_path?: string;
  systemdUnitPath?: string;
  systemd_unit_path?: string;
  configDir?: string;
  config_dir?: string;
  reloadStatus?: string;
  reload_status?: string;
  reloadInfo?: string;
  reload_info?: string;
  restartStatus?: string;
  restart_status?: string;
  restartInfo?: string;
  restart_info?: string;
  healthStatus?: string;
  health_status?: string;
  healthInfo?: string;
  health_info?: string;
  configPath?: string;
  config_path?: string;
  path?: string;
  rollbackAvailable?: boolean | string | number;
  rollback_available?: boolean | string | number;
  canRollback?: boolean | string | number;
  can_rollback?: boolean | string | number;
  phases?: RuntimeApplyPhase[];
  stages?: RuntimeApplyPhase[];
  steps?: RuntimeApplyPhase[];
};

export type RuntimeRef =
  | Runtime
  | string
  | {
      name?: string;
      type?: Runtime | string;
      capabilities?: string[];
      runtimeCapabilities?: string[];
      runtime_capabilities?: string[];
      version?: string;
      runtimeVersion?: string;
      runtime_version?: string;
      serviceStatus?: RuntimeServiceStatus;
      service_status?: RuntimeServiceStatus;
      serviceMode?: RuntimeServiceMode;
      service_mode?: RuntimeServiceMode;
      runtimeManaged?: boolean | string | number;
      runtime_managed?: boolean | string | number;
      managedByOuUi?: boolean | string | number;
      managedByOuui?: boolean | string | number;
      managed_by_ou_ui?: boolean | string | number;
      unitPath?: string;
      unit_path?: string;
      systemdUnitPath?: string;
      systemd_unit_path?: string;
      configDir?: string;
      config_dir?: string;
      reloadStatus?: string;
      reload_status?: string;
      restartStatus?: string;
      restart_status?: string;
      healthStatus?: string;
      health_status?: string;
      configPath?: string;
      config_path?: string;
      rollbackAvailable?: boolean | string | number;
      rollback_available?: boolean | string | number;
      apply?: RuntimeApplySnapshot;
      runtimeApply?: RuntimeApplySnapshot;
      runtime_apply?: RuntimeApplySnapshot;
    };

export type Protocol = "VLESS Reality" | "VMess" | "Trojan" | "Shadowsocks" | "Hysteria2";

export type ControlTaskStatus = "pending" | "running" | "success" | "failed";
export type LegacyTaskStatus = "queued" | "done";

export type AgentControlTask = {
  id?: string;
  status?: ControlTaskStatus | LegacyTaskStatus | string;
  state?: ControlTaskStatus | LegacyTaskStatus | string;
  stage?: RuntimeApplyStage | string;
  applyStage?: RuntimeApplyStage | string;
  apply_stage?: RuntimeApplyStage | string;
  currentStage?: RuntimeApplyStage | string;
  current_stage?: RuntimeApplyStage | string;
  failureStage?: RuntimeApplyStage | string;
  failure_stage?: RuntimeApplyStage | string;
  failedStage?: RuntimeApplyStage | string;
  failed_stage?: RuntimeApplyStage | string;
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

export type HostTuneStage =
  | "detect"
  | "apply"
  | "sysctl"
  | "bbr"
  | "install"
  | "verify";

export type HostTunePhase = {
  stage?: HostTuneStage | string;
  name?: HostTuneStage | string;
  phase?: HostTuneStage | string;
  status?: ControlTaskStatus | LegacyTaskStatus | string;
  state?: ControlTaskStatus | LegacyTaskStatus | string;
};

export type HostTuningSnapshot = {
  taskId?: string;
  task_id?: string;
  status?: ControlTaskStatus | LegacyTaskStatus | string;
  state?: ControlTaskStatus | LegacyTaskStatus | string;
  currentStage?: HostTuneStage | string;
  current_stage?: HostTuneStage | string;
  stage?: HostTuneStage | string;
  failureStage?: HostTuneStage | string;
  failure_stage?: HostTuneStage | string;
  failedStage?: HostTuneStage | string;
  failed_stage?: HostTuneStage | string;
  bbrStatus?: string;
  bbr_status?: string;
  bbrVersion?: string;
  bbr_version?: string;
  sysctlProfile?: string;
  sysctl_profile?: string;
  rebootRequired?: boolean | string | number;
  reboot_required?: boolean | string | number;
  kernel?: string;
  kernelVersion?: string;
  kernel_version?: string;
  currentCongestionControl?: string;
  current_congestion_control?: string;
  congestionControl?: string;
  congestion_control?: string;
  targetCongestionControl?: string;
  target_congestion_control?: string;
  eta?: string;
  updatedAt?: string;
  updated_at?: string;
  phases?: HostTunePhase[];
  stages?: HostTunePhase[];
  steps?: HostTunePhase[];
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
  runtimeVersion?: string;
  runtime_version?: string;
  serviceStatus?: RuntimeServiceStatus;
  service_status?: RuntimeServiceStatus;
  serviceMode?: RuntimeServiceMode;
  service_mode?: RuntimeServiceMode;
  runtimeManaged?: boolean | string | number;
  runtime_managed?: boolean | string | number;
  managedByOuUi?: boolean | string | number;
  managedByOuui?: boolean | string | number;
  managed_by_ou_ui?: boolean | string | number;
  unitPath?: string;
  unit_path?: string;
  systemdUnitPath?: string;
  systemd_unit_path?: string;
  configDir?: string;
  config_dir?: string;
  reloadStatus?: string;
  reload_status?: string;
  reloadInfo?: string;
  reload_info?: string;
  restartStatus?: string;
  restart_status?: string;
  restartInfo?: string;
  restart_info?: string;
  healthStatus?: string;
  health_status?: string;
  healthInfo?: string;
  health_info?: string;
  configPath?: string;
  config_path?: string;
  rollbackAvailable?: boolean | string | number;
  rollback_available?: boolean | string | number;
  applyStage?: RuntimeApplyStage | string;
  apply_stage?: RuntimeApplyStage | string;
  currentStage?: RuntimeApplyStage | string;
  current_stage?: RuntimeApplyStage | string;
  failureStage?: RuntimeApplyStage | string;
  failure_stage?: RuntimeApplyStage | string;
  failedStage?: RuntimeApplyStage | string;
  failed_stage?: RuntimeApplyStage | string;
  runtimeApply?: RuntimeApplySnapshot;
  runtime_apply?: RuntimeApplySnapshot;
  apply?: RuntimeApplySnapshot;
  failureReason?: string;
  failure_reason?: string;
  retryCount?: number;
  retry_count?: number;
  retries?: number;
  hostTuning?: HostTuningSnapshot;
  host_tuning?: HostTuningSnapshot;
  networkOptimization?: HostTuningSnapshot;
  network_optimization?: HostTuningSnapshot;
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
  stage?: RuntimeApplyStage | string;
  applyStage?: RuntimeApplyStage | string;
  apply_stage?: RuntimeApplyStage | string;
  currentStage?: RuntimeApplyStage | string;
  current_stage?: RuntimeApplyStage | string;
  failureStage?: RuntimeApplyStage | string;
  failure_stage?: RuntimeApplyStage | string;
  failedStage?: RuntimeApplyStage | string;
  failed_stage?: RuntimeApplyStage | string;
  runtimeVersion?: string;
  runtime_version?: string;
  serviceStatus?: RuntimeServiceStatus;
  service_status?: RuntimeServiceStatus;
  serviceMode?: RuntimeServiceMode;
  service_mode?: RuntimeServiceMode;
  runtimeManaged?: boolean | string | number;
  runtime_managed?: boolean | string | number;
  managedByOuUi?: boolean | string | number;
  managedByOuui?: boolean | string | number;
  managed_by_ou_ui?: boolean | string | number;
  unitPath?: string;
  unit_path?: string;
  systemdUnitPath?: string;
  systemd_unit_path?: string;
  configDir?: string;
  config_dir?: string;
  reloadStatus?: string;
  reload_status?: string;
  reloadInfo?: string;
  reload_info?: string;
  restartStatus?: string;
  restart_status?: string;
  restartInfo?: string;
  restart_info?: string;
  healthStatus?: string;
  health_status?: string;
  healthInfo?: string;
  health_info?: string;
  configPath?: string;
  config_path?: string;
  rollbackAvailable?: boolean | string | number;
  rollback_available?: boolean | string | number;
  runtimeApply?: RuntimeApplySnapshot;
  runtime_apply?: RuntimeApplySnapshot;
  apply?: RuntimeApplySnapshot;
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
    runtimeVersion: "Xray 1.8.24",
    serviceStatus: "reloading",
    serviceMode: "managed",
    runtimeManaged: true,
    unitPath: "/etc/systemd/system/ou-runtime@ou-hkg-01.service",
    configDir: "/etc/ou/runtime/xray-hkg-01",
    configPath: "/etc/ou/runtime/xray-hkg-01.json",
    rollbackAvailable: true,
    runtimeApply: {
      currentStage: "reload",
      runtimeVersion: "Xray 1.8.24",
      serviceStatus: "reloading",
      serviceMode: "managed",
      runtimeManaged: true,
      unitPath: "/etc/systemd/system/ou-runtime@ou-hkg-01.service",
      configDir: "/etc/ou/runtime/xray-hkg-01",
      configPath: "/etc/ou/runtime/xray-hkg-01.json",
      reloadStatus: "running",
      reloadInfo: "systemd reload queued after config render",
      restartStatus: "idle",
      restartInfo: "no restart required for rolling delivery",
      healthStatus: "checking",
      healthInfo: "TCP dial and handshake probes in progress",
      rollbackAvailable: true,
      phases: [
        { stage: "render", status: "success" },
        { stage: "install", status: "success" },
        { stage: "apply", status: "success" },
        { stage: "reload", status: "running" },
        { stage: "health", status: "pending" },
        { stage: "rollback", status: "pending" }
      ]
    },
    controlTask: {
      status: "running",
      currentStage: "reload",
      retryCount: 0
    },
    hostTuning: {
      taskId: "tune-91021",
      status: "running",
      currentStage: "bbr",
      bbrStatus: "BBR v3 candidate active",
      bbrVersion: "bbr3",
      sysctlProfile: "edge-throughput-v3",
      rebootRequired: false,
      kernelVersion: "6.8.12-ou1",
      currentCongestionControl: "bbr",
      targetCongestionControl: "bbr",
      eta: "48s",
      updatedAt: "18s ago",
      phases: [
        { stage: "detect", status: "success" },
        { stage: "apply", status: "success" },
        { stage: "sysctl", status: "success" },
        { stage: "bbr", status: "running" },
        { stage: "install", status: "pending" },
        { stage: "verify", status: "pending" }
      ]
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
    runtimeVersion: "Hysteria2 2.6.1",
    serviceStatus: "running",
    serviceMode: "external",
    runtimeManaged: false,
    unitPath: "/opt/ou-runtime/hysteria2-sin-02.service",
    configDir: "/opt/ou-runtime/hysteria2-sin-02",
    configPath: "/etc/ou/runtime/hysteria2-sin-02.yaml",
    rollbackAvailable: true,
    runtimeApply: {
      currentStage: "render",
      runtimeVersion: "Hysteria2 2.6.1",
      serviceStatus: "running",
      service_mode: "external",
      runtime_managed: false,
      systemd_unit_path: "/opt/ou-runtime/hysteria2-sin-02.service",
      config_dir: "/opt/ou-runtime/hysteria2-sin-02",
      configPath: "/etc/ou/runtime/hysteria2-sin-02.yaml",
      reloadStatus: "idle",
      reloadInfo: "managed by external supervisor",
      restartStatus: "available",
      restartInfo: "restart delegated to node operator",
      healthStatus: "healthy",
      healthInfo: "probe window stable over last 12 checks",
      rollbackAvailable: true,
      phases: [
        { stage: "render", status: "running" },
        { stage: "install", status: "pending" },
        { stage: "apply", status: "pending" },
        { stage: "reload", status: "pending" },
        { stage: "health", status: "pending" },
        { stage: "rollback", status: "pending" }
      ]
    },
    controlTask: {
      status: "pending",
      currentStage: "render",
      retryCount: 1
    },
    networkOptimization: {
      taskId: "tune-91022",
      state: "pending",
      currentStage: "detect",
      bbrStatus: "BBR available",
      bbrVersion: "bbr",
      sysctlProfile: "udp-transit-balanced",
      rebootRequired: false,
      kernel: "6.6.28-cloud",
      current_congestion_control: "cubic",
      target_congestion_control: "bbr",
      eta: "2m 10s",
      updatedAt: "42s ago",
      stages: [
        { stage: "detect", status: "pending" },
        { stage: "apply", status: "pending" },
        { stage: "sysctl", status: "pending" },
        { stage: "bbr", status: "pending" },
        { stage: "install", status: "pending" },
        { stage: "verify", status: "pending" }
      ]
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
    runtimeVersion: "Xray 1.8.23",
    serviceStatus: "degraded",
    serviceMode: "managed",
    runtimeManaged: true,
    unitPath: "/etc/systemd/system/ou-runtime@ou-tyo-03.service",
    configDir: "/etc/ou/runtime/xray-tyo-03",
    configPath: "/etc/ou/runtime/xray-tyo-03.json",
    rollbackAvailable: true,
    failureStage: "health",
    runtimeApply: {
      currentStage: "health",
      failureStage: "health",
      runtimeVersion: "Xray 1.8.23",
      serviceStatus: "degraded",
      serviceMode: "managed",
      runtimeManaged: true,
      unitPath: "/etc/systemd/system/ou-runtime@ou-tyo-03.service",
      configDir: "/etc/ou/runtime/xray-tyo-03",
      configPath: "/etc/ou/runtime/xray-tyo-03.json",
      reloadStatus: "completed",
      reloadInfo: "reload succeeded before health regression",
      restartStatus: "blocked",
      restartInfo: "restart held due to certificate validation failure",
      healthStatus: "failed",
      healthInfo: "health probe timed out on certificate chain validation",
      rollbackAvailable: true,
      phases: [
        { stage: "render", status: "success" },
        { stage: "install", status: "success" },
        { stage: "apply", status: "success" },
        { stage: "reload", status: "success" },
        { stage: "health", status: "failed" },
        { stage: "rollback", status: "pending" }
      ]
    },
    controlTask: {
      status: "failed",
      failedStage: "health",
      failureReason: "Certificate chain precheck failed",
      retryCount: 2
    },
    hostTuning: {
      task_id: "tune-91017",
      status: "failed",
      current_stage: "install",
      failedStage: "install",
      bbr_status: "BBR v3 module install blocked",
      bbr_version: "bbr3",
      sysctl_profile: "relay-low-latency",
      reboot_required: true,
      kernel_version: "5.15.0-107-generic",
      congestion_control: "cubic",
      targetCongestionControl: "bbr",
      eta: "manual review",
      updated_at: "2m ago",
      phases: [
        { stage: "detect", status: "success" },
        { stage: "apply", status: "success" },
        { stage: "sysctl", status: "success" },
        { stage: "bbr", status: "success" },
        { stage: "install", status: "failed" },
        { stage: "verify", status: "pending" }
      ]
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
    runtimeVersion: "Hysteria2 2.5.2",
    serviceStatus: "maintenance",
    serviceMode: "external",
    runtimeManaged: false,
    unitPath: "/opt/ou-runtime/hysteria2-lax-04.service",
    configDir: "/opt/ou-runtime/hysteria2-lax-04",
    configPath: "/etc/ou/runtime/hysteria2-lax-04.yaml",
    rollbackAvailable: false,
    runtimeApply: {
      currentStage: "rollback",
      runtimeVersion: "Hysteria2 2.5.2",
      serviceStatus: "maintenance",
      serviceMode: "external",
      runtimeManaged: false,
      unitPath: "/opt/ou-runtime/hysteria2-lax-04.service",
      configDir: "/opt/ou-runtime/hysteria2-lax-04",
      configPath: "/etc/ou/runtime/hysteria2-lax-04.yaml",
      reloadStatus: "idle",
      reloadInfo: "external service left untouched",
      restartStatus: "idle",
      restartInfo: "operator will restart outside OU-UI",
      healthStatus: "idle",
      healthInfo: "maintenance window keeps health checks paused",
      rollbackAvailable: false,
      phases: [
        { stage: "render", status: "success" },
        { stage: "install", status: "success" },
        { stage: "apply", status: "success" },
        { stage: "reload", status: "success" },
        { stage: "health", status: "success" },
        { stage: "rollback", status: "success" }
      ]
    },
    controlTask: {
      status: "success",
      currentStage: "rollback",
      retryCount: 0
    },
    host_tuning: {
      taskId: "tune-90988",
      status: "success",
      currentStage: "verify",
      bbrStatus: "BBR active",
      bbrVersion: "bbr",
      sysctlProfile: "maintenance-baseline",
      rebootRequired: true,
      kernelVersion: "6.1.82-lts",
      currentCongestionControl: "bbr",
      targetCongestionControl: "bbr",
      eta: "done",
      updatedAt: "maintenance window",
      phases: [
        { stage: "detect", status: "success" },
        { stage: "apply", status: "success" },
        { stage: "sysctl", status: "success" },
        { stage: "bbr", status: "success" },
        { stage: "install", status: "success" },
        { stage: "verify", status: "success" }
      ]
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
    currentStage: "reload",
    runtimeVersion: "Xray 1.8.24",
    serviceStatus: "reloading",
    serviceMode: "managed",
    runtimeManaged: true,
    unitPath: "/etc/systemd/system/ou-runtime@ou-hkg-01.service",
    configDir: "/etc/ou/runtime/xray-hkg-01",
    configPath: "/etc/ou/runtime/xray-hkg-01.json",
    reloadStatus: "running",
    reloadInfo: "systemd reload queued after config render",
    restartStatus: "idle",
    restartInfo: "rolling delivery avoids restart",
    healthStatus: "checking",
    healthInfo: "handshake probes warming up",
    rollbackAvailable: true,
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
    currentStage: "render",
    runtimeVersion: "Hysteria2 2.6.1",
    serviceStatus: "running",
    serviceMode: "external",
    runtimeManaged: false,
    unitPath: "/opt/ou-runtime/hysteria2-sin-02.service",
    configDir: "/opt/ou-runtime/hysteria2-sin-02",
    configPath: "/etc/ou/runtime/hysteria2-sin-02.yaml",
    reloadStatus: "idle",
    reloadInfo: "external supervisor controls reload",
    restartStatus: "available",
    restartInfo: "restart action delegated",
    healthStatus: "healthy",
    healthInfo: "UDP probe success rate above threshold",
    rollbackAvailable: true,
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
    currentStage: "health",
    runtimeVersion: "Xray 1.8.23",
    serviceStatus: "running",
    serviceMode: "managed",
    runtimeManaged: true,
    unitPath: "/etc/systemd/system/ou-runtime@ou-tyo-03.service",
    configDir: "/etc/ou/runtime/xray-tyo-03",
    configPath: "/etc/ou/runtime/xray-tyo-03.json",
    reloadStatus: "completed",
    reloadInfo: "reload completed in 1.4s",
    restartStatus: "idle",
    restartInfo: "restart not needed after reload",
    healthStatus: "healthy",
    healthInfo: "TLS and inbound probes passed",
    rollbackAvailable: true,
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
    currentStage: "health",
    failedStage: "health",
    runtimeVersion: "Xray 1.8.23",
    serviceStatus: "degraded",
    serviceMode: "managed",
    runtimeManaged: true,
    unitPath: "/etc/systemd/system/ou-runtime@ou-tyo-03.service",
    configDir: "/etc/ou/runtime/xray-tyo-03",
    configPath: "/etc/ou/runtime/xray-tyo-03.json",
    reloadStatus: "completed",
    reloadInfo: "reload completed before health failure",
    restartStatus: "blocked",
    restartInfo: "restart held by failed health gate",
    healthStatus: "failed",
    healthInfo: "certificate chain validation failed",
    rollbackAvailable: true,
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

import type {
  Agent,
  ControlTaskStatus,
  DeployTask,
  RuntimeApplySnapshot,
  RuntimeApplyStage,
  RuntimeRef
} from "./data";

export type ControlTone = "ok" | "warning" | "danger" | "muted" | "info";

export type ControlBadgeState = {
  label: string;
  tone: ControlTone;
};

export type ControlTaskView = {
  id?: string;
  status: ControlTaskStatus;
  failureReason?: string;
  retryCount: number;
};

export type RuntimeApplyPhaseView = {
  stage: RuntimeApplyStage;
  status: ControlTaskStatus;
};

export type RuntimeApplyView = {
  currentStage: RuntimeApplyStage;
  runtimeVersion: string;
  serviceStatus: string;
  serviceMode: string;
  runtimeManaged: boolean;
  unitPath: string;
  configDir: string;
  configPath: string;
  reloadStatus: string;
  reloadInfo: string;
  restartStatus: string;
  restartInfo: string;
  healthStatus: string;
  healthInfo: string;
  rollbackAvailable: boolean;
  failureStage?: RuntimeApplyStage;
  phases: RuntimeApplyPhaseView[];
};

export const runtimeApplyStages: RuntimeApplyStage[] = [
  "render",
  "install",
  "apply",
  "reload",
  "health",
  "rollback"
];

export const runtimeApplyStageLabel: Record<RuntimeApplyStage, string> = {
  render: "Render",
  install: "Install",
  apply: "Apply",
  reload: "Reload",
  health: "Health",
  rollback: "Rollback"
};

const taskStatusMap: Record<string, ControlTaskStatus> = {
  active: "running",
  applied: "success",
  complete: "success",
  completed: "success",
  done: "success",
  error: "failed",
  errored: "failed",
  failed: "failed",
  failure: "failed",
  inflight: "running",
  pending: "pending",
  processing: "running",
  queued: "pending",
  rejected: "failed",
  running: "running",
  scheduled: "pending",
  success: "success",
  succeeded: "success",
  timeout: "failed",
  waiting: "pending"
};

const runtimeApplyStageMap: Record<string, RuntimeApplyStage> = {
  apply: "apply",
  apply_config: "apply",
  applyconfig: "apply",
  config: "apply",
  configure: "apply",
  health: "health",
  health_check: "health",
  healthcheck: "health",
  install: "install",
  install_runtime: "install",
  installruntime: "install",
  precheck: "install",
  runtime_install: "install",
  runtimeinstall: "install",
  probe: "health",
  probes: "health",
  reload: "reload",
  reload_runtime: "reload",
  reloadruntime: "reload",
  render: "render",
  render_config: "render",
  renderconfig: "render",
  rollback: "rollback",
  roll_back: "rollback",
  revert: "rollback"
};

export function getRegistrationState(agent: Agent): ControlBadgeState {
  const explicit = pickString(
    agent.registrationStatus,
    agent.registration_status,
    agent.registerStatus,
    agent.registrationState,
    agent.registration_state
  );

  if (explicit) {
    return {
      label: formatToken(explicit),
      tone: registrationTone(explicit)
    };
  }

  const registered = pickBoolean(agent.registered, agent.isRegistered, agent.is_registered);
  if (registered !== undefined) {
    return registered
      ? { label: "Registered", tone: "ok" }
      : { label: "Unregistered", tone: "muted" };
  }

  return { label: "Unknown", tone: "muted" };
}

export function getAuthState(agent: Agent): ControlBadgeState {
  const explicit = pickString(
    agent.authStatus,
    agent.auth_status,
    agent.authenticationStatus,
    agent.authentication_status
  );

  if (explicit) {
    return {
      label: formatToken(explicit),
      tone: authTone(explicit)
    };
  }

  const authenticated = pickBoolean(
    agent.authenticated,
    agent.isAuthenticated,
    agent.is_authenticated
  );
  if (authenticated !== undefined) {
    return authenticated
      ? { label: "Authenticated", tone: "ok" }
      : { label: "Unauthenticated", tone: "danger" };
  }

  return { label: "Unknown", tone: "muted" };
}

export function getLastHeartbeat(agent: Agent): string {
  return (
    pickString(
      agent.lastHeartbeat,
      agent.lastHeartbeatAt,
      agent.last_heartbeat,
      agent.last_heartbeat_at,
      agent.heartbeatAt,
      agent.heartbeat_at,
      agent.updatedAt
    ) ?? "No heartbeat"
  );
}

export function getRuntimeLabel(runtime: RuntimeRef | undefined): string {
  if (typeof runtime === "string") {
    return runtime.trim() || "Unknown runtime";
  }

  if (isRecord(runtime)) {
    return (
      pickString(
        readField(runtime, "name"),
        readField(runtime, "type"),
        readField(runtime, "runtime"),
        readField(runtime, "id")
      ) ?? "Unknown runtime"
    );
  }

  return "Unknown runtime";
}

export function getRuntimeCapabilities(agent: Agent): string[] {
  const runtime = isRecord(agent.runtime) ? agent.runtime : undefined;
  const candidates = [
    agent.runtimeCapabilities,
    agent.runtime_capabilities,
    agent.capabilities,
    readField(runtime, "capabilities"),
    readField(runtime, "runtimeCapabilities"),
    readField(runtime, "runtime_capabilities")
  ];

  for (const candidate of candidates) {
    const capabilities = toStringList(candidate);
    if (capabilities.length > 0) {
      return capabilities;
    }
  }

  return [];
}

export function getAgentRuntimeApply(agent: Agent): RuntimeApplyView {
  const runtime = isRecord(agent.runtime) ? agent.runtime : undefined;
  const task = pickRecord(
    agent.controlTask,
    agent.control_task,
    agent.currentTask,
    agent.current_task,
    agent.task
  );
  const apply = pickRuntimeApplySnapshot(
    agent.runtimeApply,
    agent.runtime_apply,
    agent.apply,
    readField(runtime, "runtimeApply"),
    readField(runtime, "runtime_apply"),
    readField(runtime, "apply")
  );
  const status = normalizeTaskStatus(
    pickFirst(
      readField(task, "status"),
      readField(task, "state"),
      agent.taskStatus,
      agent.task_status
    )
  );
  const currentStage = normalizeApplyStage(
    pickFirst(
      readField(apply, "currentStage"),
      readField(apply, "current_stage"),
      readField(apply, "applyStage"),
      readField(apply, "apply_stage"),
      readField(apply, "stage"),
      readField(task, "currentStage"),
      readField(task, "current_stage"),
      readField(task, "applyStage"),
      readField(task, "apply_stage"),
      readField(task, "stage"),
      agent.currentStage,
      agent.current_stage,
      agent.applyStage,
      agent.apply_stage
    ),
    inferStageFromStatus(status)
  );
  const failureStage = normalizeOptionalApplyStage(
    pickFirst(
      readField(apply, "failureStage"),
      readField(apply, "failure_stage"),
      readField(apply, "failedStage"),
      readField(apply, "failed_stage"),
      readField(task, "failureStage"),
      readField(task, "failure_stage"),
      readField(task, "failedStage"),
      readField(task, "failed_stage"),
      agent.failureStage,
      agent.failure_stage,
      agent.failedStage,
      agent.failed_stage
    )
  );
  const runtimeManaged = pickBoolean(
    readField(apply, "runtimeManaged"),
    readField(apply, "runtime_managed"),
    readField(apply, "managedByOuUi"),
    readField(apply, "managedByOuui"),
    readField(apply, "managed_by_ou_ui"),
    agent.runtimeManaged,
    agent.runtime_managed,
    agent.managedByOuUi,
    agent.managedByOuui,
    agent.managed_by_ou_ui,
    readField(runtime, "runtimeManaged"),
    readField(runtime, "runtime_managed"),
    readField(runtime, "managedByOuUi"),
    readField(runtime, "managedByOuui"),
    readField(runtime, "managed_by_ou_ui")
  );

  return {
    currentStage,
    runtimeVersion:
      pickString(
        readField(apply, "runtimeVersion"),
        readField(apply, "runtime_version"),
        readField(apply, "version"),
        agent.runtimeVersion,
        agent.runtime_version,
        readField(runtime, "runtimeVersion"),
        readField(runtime, "runtime_version"),
        readField(runtime, "version")
      ) ?? "Version unknown",
    serviceStatus:
      pickString(
        readField(apply, "serviceStatus"),
        readField(apply, "service_status"),
        readField(apply, "service"),
        agent.serviceStatus,
        agent.service_status,
        readField(runtime, "serviceStatus"),
        readField(runtime, "service_status")
      ) ?? "unknown",
    serviceMode:
      pickString(
        readField(apply, "serviceMode"),
        readField(apply, "service_mode"),
        readField(apply, "mode"),
        agent.serviceMode,
        agent.service_mode,
        readField(runtime, "serviceMode"),
        readField(runtime, "service_mode"),
        readField(runtime, "mode")
      ) ?? "managed",
    runtimeManaged: runtimeManaged ?? true,
    unitPath:
      pickString(
        readField(apply, "unitPath"),
        readField(apply, "unit_path"),
        readField(apply, "systemdUnitPath"),
        readField(apply, "systemd_unit_path"),
        agent.unitPath,
        agent.unit_path,
        agent.systemdUnitPath,
        agent.systemd_unit_path,
        readField(runtime, "unitPath"),
        readField(runtime, "unit_path"),
        readField(runtime, "systemdUnitPath"),
        readField(runtime, "systemd_unit_path")
      ) ?? "/etc/systemd/system/ou-runtime.service",
    configDir:
      pickString(
        readField(apply, "configDir"),
        readField(apply, "config_dir"),
        agent.configDir,
        agent.config_dir,
        readField(runtime, "configDir"),
        readField(runtime, "config_dir")
      ) ?? "Not reported",
    configPath:
      pickString(
        readField(apply, "configPath"),
        readField(apply, "config_path"),
        readField(apply, "path"),
        agent.configPath,
        agent.config_path,
        readField(runtime, "configPath"),
        readField(runtime, "config_path")
      ) ?? "Not reported",
    reloadStatus:
      pickString(
        readField(apply, "reloadStatus"),
        readField(apply, "reload_status"),
        agent.reloadStatus,
        agent.reload_status,
        readField(runtime, "reloadStatus"),
        readField(runtime, "reload_status")
      ) ?? "unknown",
    reloadInfo:
      pickString(
        readField(apply, "reloadInfo"),
        readField(apply, "reload_info"),
        agent.reloadInfo,
        agent.reload_info,
        readField(runtime, "reloadInfo"),
        readField(runtime, "reload_info")
      ) ?? "No reload signal",
    restartStatus:
      pickString(
        readField(apply, "restartStatus"),
        readField(apply, "restart_status"),
        agent.restartStatus,
        agent.restart_status,
        readField(runtime, "restartStatus"),
        readField(runtime, "restart_status")
      ) ?? "unknown",
    restartInfo:
      pickString(
        readField(apply, "restartInfo"),
        readField(apply, "restart_info"),
        agent.restartInfo,
        agent.restart_info,
        readField(runtime, "restartInfo"),
        readField(runtime, "restart_info")
      ) ?? "No restart signal",
    healthStatus:
      pickString(
        readField(apply, "healthStatus"),
        readField(apply, "health_status"),
        agent.healthStatus,
        agent.health_status,
        readField(runtime, "healthStatus"),
        readField(runtime, "health_status")
      ) ?? "unknown",
    healthInfo:
      pickString(
        readField(apply, "healthInfo"),
        readField(apply, "health_info"),
        agent.healthInfo,
        agent.health_info,
        readField(runtime, "healthInfo"),
        readField(runtime, "health_info")
      ) ?? "No health signal",
    rollbackAvailable:
      pickBoolean(
        readField(apply, "rollbackAvailable"),
        readField(apply, "rollback_available"),
        readField(apply, "canRollback"),
        readField(apply, "can_rollback"),
        agent.rollbackAvailable,
        agent.rollback_available,
        readField(runtime, "rollbackAvailable"),
        readField(runtime, "rollback_available")
      ) ?? false,
    failureStage,
    phases: getRuntimeApplyPhases(apply, currentStage, status, failureStage)
  };
}

export function getDeployRuntimeApply(task: DeployTask): RuntimeApplyView {
  const runtime = isRecord(task.runtime) ? task.runtime : undefined;
  const apply = pickRuntimeApplySnapshot(
    task.runtimeApply,
    task.runtime_apply,
    task.apply,
    readField(runtime, "runtimeApply"),
    readField(runtime, "runtime_apply"),
    readField(runtime, "apply")
  );
  const status = normalizeTaskStatus(pickFirst(task.status, task.state));
  const currentStage = normalizeApplyStage(
    pickFirst(
      readField(apply, "currentStage"),
      readField(apply, "current_stage"),
      readField(apply, "applyStage"),
      readField(apply, "apply_stage"),
      readField(apply, "stage"),
      task.currentStage,
      task.current_stage,
      task.applyStage,
      task.apply_stage,
      task.stage
    ),
    inferStageFromStatus(status)
  );
  const failureStage = normalizeOptionalApplyStage(
    pickFirst(
      readField(apply, "failureStage"),
      readField(apply, "failure_stage"),
      readField(apply, "failedStage"),
      readField(apply, "failed_stage"),
      task.failureStage,
      task.failure_stage,
      task.failedStage,
      task.failed_stage
    )
  );
  const runtimeManaged = pickBoolean(
    readField(apply, "runtimeManaged"),
    readField(apply, "runtime_managed"),
    readField(apply, "managedByOuUi"),
    readField(apply, "managedByOuui"),
    readField(apply, "managed_by_ou_ui"),
    task.runtimeManaged,
    task.runtime_managed,
    task.managedByOuUi,
    task.managedByOuui,
    task.managed_by_ou_ui,
    readField(runtime, "runtimeManaged"),
    readField(runtime, "runtime_managed"),
    readField(runtime, "managedByOuUi"),
    readField(runtime, "managedByOuui"),
    readField(runtime, "managed_by_ou_ui")
  );

  return {
    currentStage,
    runtimeVersion:
      pickString(
        readField(apply, "runtimeVersion"),
        readField(apply, "runtime_version"),
        readField(apply, "version"),
        task.runtimeVersion,
        task.runtime_version,
        readField(runtime, "runtimeVersion"),
        readField(runtime, "runtime_version"),
        readField(runtime, "version")
      ) ?? "Version unknown",
    serviceStatus:
      pickString(
        readField(apply, "serviceStatus"),
        readField(apply, "service_status"),
        readField(apply, "service"),
        task.serviceStatus,
        task.service_status,
        readField(runtime, "serviceStatus"),
        readField(runtime, "service_status")
      ) ?? "unknown",
    serviceMode:
      pickString(
        readField(apply, "serviceMode"),
        readField(apply, "service_mode"),
        readField(apply, "mode"),
        task.serviceMode,
        task.service_mode,
        readField(runtime, "serviceMode"),
        readField(runtime, "service_mode"),
        readField(runtime, "mode")
      ) ?? "managed",
    runtimeManaged: runtimeManaged ?? true,
    unitPath:
      pickString(
        readField(apply, "unitPath"),
        readField(apply, "unit_path"),
        readField(apply, "systemdUnitPath"),
        readField(apply, "systemd_unit_path"),
        task.unitPath,
        task.unit_path,
        task.systemdUnitPath,
        task.systemd_unit_path,
        readField(runtime, "unitPath"),
        readField(runtime, "unit_path"),
        readField(runtime, "systemdUnitPath"),
        readField(runtime, "systemd_unit_path")
      ) ?? "/etc/systemd/system/ou-runtime.service",
    configDir:
      pickString(
        readField(apply, "configDir"),
        readField(apply, "config_dir"),
        task.configDir,
        task.config_dir,
        readField(runtime, "configDir"),
        readField(runtime, "config_dir")
      ) ?? "Not reported",
    configPath:
      pickString(
        readField(apply, "configPath"),
        readField(apply, "config_path"),
        readField(apply, "path"),
        task.configPath,
        task.config_path,
        readField(runtime, "configPath"),
        readField(runtime, "config_path")
      ) ?? "Not reported",
    reloadStatus:
      pickString(
        readField(apply, "reloadStatus"),
        readField(apply, "reload_status"),
        task.reloadStatus,
        task.reload_status,
        readField(runtime, "reloadStatus"),
        readField(runtime, "reload_status")
      ) ?? "unknown",
    reloadInfo:
      pickString(
        readField(apply, "reloadInfo"),
        readField(apply, "reload_info"),
        task.reloadInfo,
        task.reload_info,
        readField(runtime, "reloadInfo"),
        readField(runtime, "reload_info")
      ) ?? "No reload signal",
    restartStatus:
      pickString(
        readField(apply, "restartStatus"),
        readField(apply, "restart_status"),
        task.restartStatus,
        task.restart_status,
        readField(runtime, "restartStatus"),
        readField(runtime, "restart_status")
      ) ?? "unknown",
    restartInfo:
      pickString(
        readField(apply, "restartInfo"),
        readField(apply, "restart_info"),
        task.restartInfo,
        task.restart_info,
        readField(runtime, "restartInfo"),
        readField(runtime, "restart_info")
      ) ?? "No restart signal",
    healthStatus:
      pickString(
        readField(apply, "healthStatus"),
        readField(apply, "health_status"),
        task.healthStatus,
        task.health_status,
        readField(runtime, "healthStatus"),
        readField(runtime, "health_status")
      ) ?? "unknown",
    healthInfo:
      pickString(
        readField(apply, "healthInfo"),
        readField(apply, "health_info"),
        task.healthInfo,
        task.health_info,
        readField(runtime, "healthInfo"),
        readField(runtime, "health_info")
      ) ?? "No health signal",
    rollbackAvailable:
      pickBoolean(
        readField(apply, "rollbackAvailable"),
        readField(apply, "rollback_available"),
        readField(apply, "canRollback"),
        readField(apply, "can_rollback"),
        task.rollbackAvailable,
        task.rollback_available,
        readField(runtime, "rollbackAvailable"),
        readField(runtime, "rollback_available")
      ) ?? false,
    failureStage,
    phases: getRuntimeApplyPhases(apply, currentStage, status, failureStage)
  };
}

export function getAgentTaskState(agent: Agent): ControlTaskView {
  const task = pickRecord(
    agent.controlTask,
    agent.control_task,
    agent.currentTask,
    agent.current_task,
    agent.task
  );

  return {
    id: pickString(readField(task, "id")),
    status: normalizeTaskStatus(
      pickFirst(
        readField(task, "status"),
        readField(task, "state"),
        agent.taskStatus,
        agent.task_status
      )
    ),
    failureReason: pickString(
      readField(task, "failureReason"),
      readField(task, "failure_reason"),
      readField(task, "errorMessage"),
      readField(task, "error_message"),
      readField(task, "error"),
      agent.failureReason,
      agent.failure_reason
    ),
    retryCount:
      pickNumber(
        readField(task, "retryCount"),
        readField(task, "retry_count"),
        readField(task, "retries"),
        readField(task, "attempts"),
        agent.retryCount,
        agent.retry_count,
        agent.retries
      ) ?? 0
  };
}

export function getDeployTaskState(task: DeployTask): ControlTaskView {
  return {
    id: task.id,
    status: normalizeTaskStatus(pickFirst(task.status, task.state)),
    failureReason: pickString(
      task.failureReason,
      task.failure_reason,
      task.errorMessage,
      task.error_message,
      task.error
    ),
    retryCount:
      pickNumber(task.retryCount, task.retry_count, task.retries, task.attempts) ?? 0
  };
}

export function getTaskProgress(task: DeployTask): number {
  const progress = pickNumber(task.progress);

  if (progress !== undefined) {
    return clamp(progress, 0, 100);
  }

  const status = normalizeTaskStatus(pickFirst(task.status, task.state));
  if (status === "success" || status === "failed") {
    return 100;
  }
  if (status === "running") {
    return 58;
  }
  return 12;
}

export function normalizeTaskStatus(value: unknown): ControlTaskStatus {
  const raw = pickString(value)?.toLowerCase();

  if (!raw) {
    return "pending";
  }

  return taskStatusMap[raw] ?? "pending";
}

function pickFirst(...values: unknown[]): unknown {
  return values.find((value) => value !== undefined && value !== null && value !== "");
}

function pickString(...values: unknown[]): string | undefined {
  const value = pickFirst(...values);

  if (typeof value === "string") {
    const trimmed = value.trim();
    return trimmed.length > 0 ? trimmed : undefined;
  }

  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }

  return undefined;
}

function pickBoolean(...values: unknown[]): boolean | undefined {
  const value = pickFirst(...values);

  if (typeof value === "boolean") {
    return value;
  }

  if (typeof value === "string") {
    const normalized = value.trim().toLowerCase();
    if (["true", "yes", "1"].includes(normalized)) {
      return true;
    }
    if (["false", "no", "0"].includes(normalized)) {
      return false;
    }
  }

  return undefined;
}

function pickNumber(...values: unknown[]): number | undefined {
  const value = pickFirst(...values);

  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : undefined;
  }

  return undefined;
}

function pickRecord(...values: unknown[]): Record<string, unknown> | undefined {
  const value = pickFirst(...values);
  return isRecord(value) ? value : undefined;
}

function pickRuntimeApplySnapshot(...values: unknown[]): RuntimeApplySnapshot | undefined {
  return pickRecord(...values) as RuntimeApplySnapshot | undefined;
}

function readField(value: unknown, key: string): unknown {
  return isRecord(value) ? value[key] : undefined;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function toStringList(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value
      .map((item) => pickString(item))
      .filter((item): item is string => Boolean(item));
  }

  const text = pickString(value);
  if (!text) {
    return [];
  }

  return text
    .split(/[,|]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function getRuntimeApplyPhases(
  apply: RuntimeApplySnapshot | undefined,
  currentStage: RuntimeApplyStage,
  status: ControlTaskStatus,
  failureStage: RuntimeApplyStage | undefined
): RuntimeApplyPhaseView[] {
  const explicitPhases = firstList(
    readField(apply, "phases"),
    readField(apply, "stages"),
    readField(apply, "steps")
  );

  if (explicitPhases) {
    const normalized = explicitPhases
      .map((phase) => {
        const stage = normalizeOptionalApplyStage(
          pickFirst(
            readField(phase, "stage"),
            readField(phase, "name"),
            readField(phase, "phase")
          )
        );

        if (!stage) {
          return undefined;
        }

        return {
          stage,
          status: normalizeTaskStatus(
            pickFirst(readField(phase, "status"), readField(phase, "state"))
          )
        };
      })
      .filter((phase): phase is RuntimeApplyPhaseView => Boolean(phase));

    if (normalized.length > 0) {
      return runtimeApplyStages.map((stage) => {
        const explicit = normalized.find((phase) => phase.stage === stage);
        return (
          explicit ?? {
            stage,
            status: inferStageStatus(stage, currentStage, status, failureStage)
          }
        );
      });
    }
  }

  return runtimeApplyStages.map((stage) => ({
    stage,
    status: inferStageStatus(stage, currentStage, status, failureStage)
  }));
}

function firstList(...values: unknown[]): unknown[] | undefined {
  for (const value of values) {
    if (Array.isArray(value)) {
      return value;
    }
  }

  return undefined;
}

function normalizeApplyStage(value: unknown, fallback: RuntimeApplyStage): RuntimeApplyStage {
  return normalizeOptionalApplyStage(value) ?? fallback;
}

function normalizeOptionalApplyStage(value: unknown): RuntimeApplyStage | undefined {
  const raw = pickString(value)
    ?.toLowerCase()
    .replace(/[\s-]+/g, "_");

  if (!raw) {
    return undefined;
  }

  return runtimeApplyStageMap[raw] ?? undefined;
}

function inferStageFromStatus(status: ControlTaskStatus): RuntimeApplyStage {
  if (status === "success" || status === "failed") {
    return "health";
  }
  if (status === "running") {
    return "apply";
  }
  return "render";
}

function inferStageStatus(
  stage: RuntimeApplyStage,
  currentStage: RuntimeApplyStage,
  status: ControlTaskStatus,
  failureStage: RuntimeApplyStage | undefined
): ControlTaskStatus {
  if (failureStage === stage) {
    return "failed";
  }

  const currentIndex = runtimeApplyStages.indexOf(currentStage);
  const stageIndex = runtimeApplyStages.indexOf(stage);

  if (status === "success") {
    return stageIndex <= currentIndex ? "success" : "pending";
  }

  if (status === "failed") {
    if (stageIndex < currentIndex) {
      return "success";
    }

    return stageIndex === currentIndex ? "failed" : "pending";
  }

  if (status === "running") {
    if (stageIndex < currentIndex) {
      return "success";
    }

    return stageIndex === currentIndex ? "running" : "pending";
  }

  return stageIndex < currentIndex ? "success" : "pending";
}

function formatToken(value: string): string {
  return value
    .replace(/[_-]+/g, " ")
    .replace(/\s+/g, " ")
    .trim()
    .replace(/\b[a-z]/g, (letter) => letter.toUpperCase());
}

function registrationTone(value: string): ControlTone {
  const normalized = value.toLowerCase();

  if (matchesAny(normalized, ["failed", "denied", "rejected", "error"])) {
    return "danger";
  }
  if (matchesAny(normalized, ["pending", "provisioning", "registering"])) {
    return "warning";
  }
  if (matchesAny(normalized, ["unregistered", "not_registered", "not registered", "unknown"])) {
    return "muted";
  }
  if (matchesAny(normalized, ["registered", "active", "ready", "enrolled", "ok"])) {
    return "ok";
  }

  return "muted";
}

function authTone(value: string): ControlTone {
  const normalized = value.toLowerCase();

  if (
    matchesAny(normalized, [
      "failed",
      "expired",
      "unauthenticated",
      "not_authenticated",
      "not authenticated",
      "unauthorized",
      "revoked",
      "invalid",
      "denied"
    ])
  ) {
    return "danger";
  }
  if (matchesAny(normalized, ["pending", "refreshing", "rotating"])) {
    return "warning";
  }
  if (matchesAny(normalized, ["authenticated", "authorized", "valid", "active", "ok"])) {
    return "ok";
  }

  return "muted";
}

function matchesAny(value: string, terms: string[]): boolean {
  return terms.some((term) => value.includes(term));
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max);
}

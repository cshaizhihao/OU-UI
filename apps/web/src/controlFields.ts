import type {
  Agent,
  ControlTaskStatus,
  DeployTask,
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

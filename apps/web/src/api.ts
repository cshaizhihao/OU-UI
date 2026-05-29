import type { Agent, AgentStatus, RuntimeRef } from "./data";

const tokenKey = "ou-ui-panel-token";

export type SessionUser = {
  id: string;
  username: string;
  role: string;
  tenantId?: string;
};

export type OverviewDTO = {
  agentsTotal: number;
  agentsOnline: number;
  nodesTotal: number;
  version: string;
};

type LoginResponse = {
  token: string;
  expiresIn: number;
  user?: SessionUser;
};

type ListResponse<T> = {
  items: T[];
};

type BackendAgent = {
  id: string;
  name?: string;
  status?: AgentStatus | string;
  hostname?: string;
  os?: string;
  arch?: string;
  publicIp?: string;
  cpuCount?: number;
  memoryTotal?: number;
  trafficLimit?: number;
  queue?: number;
  capabilities?: string[] | Record<string, unknown>;
  lastMetrics?: Record<string, unknown>;
  lastSeenAt?: string;
  updatedAt?: string;
  version?: string;
  authStatus?: string;
};

export type DashboardDTO = {
  overview: OverviewDTO;
  agents: Agent[];
};

export function getStoredToken(): string {
  return localStorage.getItem(tokenKey) ?? "";
}

export function setStoredToken(token: string) {
  localStorage.setItem(tokenKey, token);
}

export function clearStoredToken() {
  localStorage.removeItem(tokenKey);
}

export async function login(username: string, password: string): Promise<LoginResponse> {
  const out = await request<LoginResponse>("/auth/login", {
    method: "POST",
    body: JSON.stringify({ username, password })
  });
  setStoredToken(out.token);
  return out;
}

export async function loadDashboard(): Promise<DashboardDTO> {
  const [overview, agents] = await Promise.all([
    request<OverviewDTO>("/overview"),
    request<ListResponse<BackendAgent>>("/agents")
  ]);
  return {
    overview,
    agents: agents.items.map(toAgentView)
  };
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set("Content-Type", "application/json");
  const token = getStoredToken();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  const res = await fetch(`${apiBase()}${path}`, { ...init, headers });
  if (res.status === 401) {
    clearStoredToken();
  }
  if (!res.ok) {
    const message = await readError(res);
    throw new Error(message || `API request failed with ${res.status}`);
  }
  return res.json() as Promise<T>;
}

function apiBase(): string {
  const envBase = import.meta.env.VITE_OUUI_API_BASE;
  if (envBase) {
    return String(envBase).replace(/\/$/, "");
  }
  const parts = window.location.pathname.split("/").filter(Boolean);
  if (parts.length > 0 && parts[0] !== "") {
    return `/${parts[0]}/api/v1`;
  }
  return "/ou-ui/api/v1";
}

async function readError(res: Response): Promise<string> {
  try {
    const body = await res.json();
    return String(body.error ?? body.message ?? "");
  } catch {
    return res.statusText;
  }
}

function toAgentView(agent: BackendAgent): Agent {
  const metrics = plainObject(agent.lastMetrics);
  const memoryTotal = numberValue(metrics.memoryTotal) || numberValue(agent.memoryTotal);
  const memoryUsed = numberValue(metrics.memoryUsed);
  const rxRate = numberValue(metrics.netRxRateBps);
  const txRate = numberValue(metrics.netTxRateBps);
  const rxBytes = numberValue(metrics.netRxBytes);
  const txBytes = numberValue(metrics.netTxBytes);
  const capabilities = stringArray(agent.capabilities);
  const status = normalizeStatus(agent.status);
  return {
    id: agent.id,
    name: agent.name || agent.hostname || agent.id,
    region: agent.publicIp ? `Public ${agent.publicIp}` : [agent.os, agent.arch].filter(Boolean).join(" / "),
    status,
    runtime: runtimeFromCapabilities(capabilities),
    ip: agent.publicIp || agent.hostname || "unreported",
    cpu: Math.round(numberValue(metrics.cpuPercent)),
    memory: memoryTotal > 0 ? Math.round((memoryUsed * 100) / memoryTotal) : 0,
    uplinkMbps: Math.round((txRate * 8) / 1_000_000),
    downlinkMbps: Math.round((rxRate * 8) / 1_000_000),
    usedTrafficGb: Math.round((rxBytes + txBytes) / 1024 / 1024 / 1024),
    quotaTrafficGb: Math.round(numberValue(agent.trafficLimit) / 1024 / 1024 / 1024),
    queue: numberValue(agent.queue),
    updatedAt: relativeTime(agent.lastSeenAt || agent.updatedAt),
    authStatus: agent.authStatus,
    lastHeartbeat: relativeTime(agent.lastSeenAt),
    runtimeCapabilities: capabilities,
    runtimeVersion: agent.version || "Agent runtime",
    serviceStatus: status === "online" ? "running" : status,
    serviceMode: "managed",
    runtimeManaged: true
  };
}

function normalizeStatus(status?: string): AgentStatus {
  if (status === "online" || status === "degraded" || status === "offline") {
    return status;
  }
  return "offline";
}

function runtimeFromCapabilities(capabilities: string[]): RuntimeRef {
  if (capabilities.some((item) => item.toLowerCase().includes("hysteria2"))) {
    return "Hysteria2";
  }
  return "Xray";
}

function plainObject(value: unknown): Record<string, unknown> {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, unknown>;
  }
  return {};
}

function numberValue(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}

function stringArray(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.map(String).filter(Boolean);
  }
  return [];
}

function relativeTime(value?: string): string {
  if (!value) {
    return "not seen";
  }
  const timestamp = new Date(value).getTime();
  if (!Number.isFinite(timestamp)) {
    return value;
  }
  const seconds = Math.max(0, Math.round((Date.now() - timestamp) / 1000));
  if (seconds < 60) {
    return `${seconds}s ago`;
  }
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) {
    return `${minutes}m ago`;
  }
  const hours = Math.round(minutes / 60);
  if (hours < 48) {
    return `${hours}h ago`;
  }
  return `${Math.round(hours / 24)}d ago`;
}

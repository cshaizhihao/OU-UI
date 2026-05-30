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

export type ManagedNode = {
  id: string;
  agentId: string;
  name: string;
  runtime: string;
  protocol: string;
  status: string;
  serviceStatus?: string;
  configPath?: string;
  lastError?: string;
  updatedAt?: string;
};

export type NodeTraffic = {
  nodeId: string;
  agentId: string;
  rxBytes: number;
  txBytes: number;
  rxRateBps: number;
  txRateBps: number;
  connections: number;
  collectedAt: string;
};

export type RoutingRule = {
  id: string;
  name: string;
  enabled: boolean;
  priority: number;
  ruleType: string;
  match: string;
  protocol?: string;
  action: string;
  targetTag?: string;
  description?: string;
};

export type LoadBalancerGroup = {
  id: string;
  name: string;
  entryTag: string;
  strategy: string;
  members: Array<Record<string, unknown>>;
  status: string;
  lastDecision?: Record<string, unknown>;
  healthCheckInterval: number;
};

export type WebhookEndpoint = {
  id: string;
  name: string;
  kind: string;
  url?: string;
  chatId?: string;
  enabled: boolean;
  eventTypes?: string[];
};

export type AlertEvent = {
  id: string;
  severity: string;
  sourceType: string;
  sourceId: string;
  eventType: string;
  message: string;
  delivered: boolean;
  lastError?: string;
  createdAt: string;
};

export type ExternalSubscription = {
  id: string;
  name: string;
  url: string;
  format: string;
  enabled: boolean;
  lastFetchedAt?: string;
  lastError?: string;
};

export type ExternalNode = {
  id: string;
  subscriptionId: string;
  name: string;
  protocol: string;
  address: string;
  port: number;
  source: string;
  enabled: boolean;
  latencyMs?: number;
  lossPercent?: number;
};

export type ClashProfile = {
  id: string;
  name: string;
  generatedYaml?: string;
  proxyGroups?: Array<Record<string, unknown>>;
  routingRules?: string[];
  ruleProviders?: Array<Record<string, unknown>>;
  updatedAt?: string;
};

export type AggregateSubscriptionFormat = "clash" | "v2ray" | "raw" | "sing-box";

export type Tenant = {
  id: string;
  name: string;
  status: string;
  role: string;
  nodeAccess?: string[];
  monthlyTrafficQuota?: number;
  perNodeTrafficQuota?: number;
  maxConnections?: number;
};

export type PanelUser = {
  id: string;
  tenantId?: string;
  username: string;
  role: string;
  status: string;
  nodeAccess?: string[];
  monthlyTrafficQuota?: number;
  perNodeTrafficQuota?: number;
  maxConnections?: number;
};

export type APIKeyCreateResponse = {
  item: {
    id: string;
    name: string;
    status: string;
    tenantId?: string;
    scopes?: string[];
    expiresAt?: string;
    lastUsedAt?: string;
  };
  apiKey: string;
};

export type APIKey = {
  id: string;
  tenantId?: string;
  name: string;
  scopes?: string[];
  status: string;
  expiresAt?: string;
  lastUsedAt?: string;
  lastUsedIp?: string;
  lastUsedUserAgent?: string;
  createdAt?: string;
  updatedAt?: string;
};

export type CopilotIncident = {
  id: string;
  question: string;
  answer: string;
  model: string;
  status: string;
  createdAt: string;
};

export type ControlTask = {
  id: string;
  agentId: string;
  type: string;
  status: string;
  attempts: number;
  lastError?: string;
  createdAt?: string;
  updatedAt?: string;
};

export type ControlPlaneDTO = {
  nodes: ManagedNode[];
  traffic: NodeTraffic[];
  routingRules: RoutingRule[];
  loadBalancers: LoadBalancerGroup[];
  webhooks: WebhookEndpoint[];
  alerts: AlertEvent[];
  subscriptions: ExternalSubscription[];
  externalNodes: ExternalNode[];
  clashProfiles: ClashProfile[];
  tenants: Tenant[];
  users: PanelUser[];
  copilotIncidents: CopilotIncident[];
  tasks: ControlTask[];
};

export type DashboardDTO = {
  overview: OverviewDTO;
  agents: Agent[];
  control: ControlPlaneDTO;
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
  const [overview, agents, control] = await Promise.all([
    request<OverviewDTO>("/overview"),
    request<ListResponse<BackendAgent>>("/agents"),
    loadControlPlane()
  ]);
  return {
    overview,
    agents: agents.items.map(toAgentView),
    control
  };
}

export async function loadControlPlane(): Promise<ControlPlaneDTO> {
  const [
    nodes,
    traffic,
    routingRules,
    loadBalancers,
    webhooks,
    alerts,
    subscriptions,
    externalNodes,
    clashProfiles,
    tenants,
    users,
    copilotIncidents,
    tasks
  ] = await Promise.all([
    request<ListResponse<ManagedNode>>("/nodes"),
    request<ListResponse<NodeTraffic>>("/traffic/nodes"),
    request<ListResponse<RoutingRule>>("/routing/rules"),
    request<ListResponse<LoadBalancerGroup>>("/load-balancers"),
    request<ListResponse<WebhookEndpoint>>("/webhooks"),
    request<ListResponse<AlertEvent>>("/alerts"),
    request<ListResponse<ExternalSubscription>>("/subscriptions"),
    request<ListResponse<ExternalNode>>("/external-nodes"),
    request<ListResponse<ClashProfile>>("/clash/profiles"),
    request<ListResponse<Tenant>>("/tenants"),
    request<ListResponse<PanelUser>>("/users"),
    request<ListResponse<CopilotIncident>>("/copilot/incidents"),
    request<ListResponse<ControlTask>>("/tasks")
  ]);
  return {
    nodes: nodes.items,
    traffic: traffic.items,
    routingRules: routingRules.items,
    loadBalancers: loadBalancers.items,
    webhooks: webhooks.items,
    alerts: alerts.items,
    subscriptions: subscriptions.items,
    externalNodes: externalNodes.items,
    clashProfiles: clashProfiles.items,
    tenants: tenants.items,
    users: users.items,
    copilotIncidents: copilotIncidents.items,
    tasks: tasks.items
  };
}

export async function optimizeAgent(agentId: string) {
  return request<{ task: ControlTask }>(`/agents/${agentId}/network-optimization`, {
    method: "POST",
    body: JSON.stringify({ profile: "bbr-v3", allowKernelInstall: true, rebootPolicy: "manual", persist: true })
  });
}

export async function createRoutingRule(input: Omit<RoutingRule, "id">) {
  return request<RoutingRule>("/routing/rules", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function applyRouting(agentIds: string[]) {
  return request<{ count: number; tasks: ControlTask[] }>("/routing/apply", {
    method: "POST",
    body: JSON.stringify({ agentIds })
  });
}

export async function createLoadBalancer(input: {
  name: string;
  entryTag: string;
  strategy: string;
  members: Array<Record<string, unknown>>;
  healthCheckInterval: number;
}) {
  return request<LoadBalancerGroup>("/load-balancers", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function createWebhook(input: {
  name: string;
  kind: string;
  url: string;
  secret?: string;
  chatId?: string;
  enabled: boolean;
  eventTypes: string[];
}) {
  return request<WebhookEndpoint>("/webhooks", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function testWebhook(id: string) {
  return request<{ ok: boolean }>(`/webhooks/${id}/test`, {
    method: "POST"
  });
}

export async function createSubscription(input: {
  name: string;
  url?: string;
  format?: string;
  content?: string;
  enabled: boolean;
}) {
  return request<ExternalSubscription>("/subscriptions", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function importSubscription(id: string, content = "") {
  return request<{ imported: number; items: ExternalNode[] }>(`/subscriptions/${id}/import`, {
    method: "POST",
    body: JSON.stringify({ content })
  });
}

export async function createClashProfile(input: {
  name: string;
  ruleProviders: Array<Record<string, unknown>>;
  proxyGroups: Array<Record<string, unknown>>;
  routingRules: string[];
  selectedNodes?: string[];
}) {
  return request<ClashProfile>("/clash/profiles", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function aggregateSubscriptionURL(format: AggregateSubscriptionFormat = "clash"): string {
  return `${apiBase()}/subscriptions/aggregate?format=${encodeURIComponent(format)}`;
}

export function clashProfileURL(id: string): string {
  return `${apiBase()}/clash/profiles/${encodeURIComponent(id)}.yaml`;
}

export async function loadAggregateSubscription(format: AggregateSubscriptionFormat = "clash"): Promise<string> {
  const headers = new Headers();
  const token = getStoredToken();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  const res = await fetch(aggregateSubscriptionURL(format), { headers });
  if (res.status === 401) {
    clearStoredToken();
  }
  if (!res.ok) {
    const message = await readError(res);
    throw new Error(message || `API request failed with ${res.status}`);
  }
  return res.text();
}

export async function createTenant(input: {
  name: string;
  status: string;
  role: string;
  nodeAccess: string[];
  monthlyTrafficQuota: number;
  perNodeTrafficQuota: number;
  maxConnections: number;
}) {
  return request<Tenant>("/tenants", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function updateTenant(
  id: string,
  input: Partial<{
    name: string;
    status: string;
    role: string;
    nodeAccess: string[];
    monthlyTrafficQuota: number;
    perNodeTrafficQuota: number;
    maxConnections: number;
  }>
) {
  return request<Tenant>(`/tenants/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: JSON.stringify(input)
  });
}

export async function createPanelUser(input: {
  tenantId: string;
  username: string;
  password: string;
  role: string;
  status: string;
  nodeAccess: string[];
  monthlyTrafficQuota: number;
  perNodeTrafficQuota: number;
  maxConnections: number;
}) {
  return request<PanelUser>("/users", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function updatePanelUser(
  id: string,
  input: Partial<{
    tenantId: string;
    username: string;
    password: string;
    role: string;
    status: string;
    nodeAccess: string[];
    monthlyTrafficQuota: number;
    perNodeTrafficQuota: number;
    maxConnections: number;
  }>
) {
  return request<PanelUser>(`/users/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: JSON.stringify(input)
  });
}

export async function createAPIKey(input: {
  tenantId: string;
  name: string;
  scopes: string[];
  status: string;
  expiresAt?: string;
}) {
  return request<APIKeyCreateResponse>("/api-keys", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function listAPIKeys() {
  return request<ListResponse<APIKey>>("/api-keys");
}

export async function updateAPIKey(
  id: string,
  input: Partial<{
    tenantId: string;
    name: string;
    scopes: string[];
    status: string;
    expiresAt: string;
  }>
) {
  return request<APIKey>(`/api-keys/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: JSON.stringify(input)
  });
}

export async function revokeAPIKey(id: string) {
  return request<{ ok: boolean; item: APIKey }>(`/api-keys/${encodeURIComponent(id)}`, {
    method: "DELETE"
  });
}

export async function askCopilot(question: string) {
  return request<CopilotIncident>("/copilot/ask", {
    method: "POST",
    body: JSON.stringify({ question })
  });
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
  const diskUsed = firstNumber(metrics.diskUsed, metrics.diskUsedBytes, metrics.disk_used_bytes);
  const diskTotal = firstNumber(metrics.diskTotal, metrics.diskTotalBytes, metrics.disk_total_bytes);
  const uptimeSeconds = numberValue(metrics.uptimeSeconds);
  const latencyMs = optionalNumber(metrics.latencyMs ?? metrics.latency_ms);
  const lossPercent = optionalNumber(metrics.lossPercent ?? metrics.loss_percent);
  const rxRate = numberValue(metrics.netRxRateBps);
  const txRate = numberValue(metrics.netTxRateBps);
  const rxBytes = numberValue(metrics.netRxBytes);
  const txBytes = numberValue(metrics.netTxBytes);
  const capabilities = stringArray(agent.capabilities);
  const status = normalizeStatus(agent.status);
  const lastSeenAt = agent.lastSeenAt || agent.updatedAt || "";
  return {
    id: agent.id,
    name: agent.name || agent.hostname || agent.id,
    region: agent.publicIp ? `Public ${agent.publicIp}` : [agent.os, agent.arch].filter(Boolean).join(" / "),
    status,
    runtime: runtimeFromCapabilities(capabilities),
    ip: agent.publicIp || agent.hostname || "unreported",
    cpuCores: numberValue(agent.cpuCount) || undefined,
    cpu: Math.round(numberValue(metrics.cpuPercent)),
    memory: memoryTotal > 0 ? Math.round((memoryUsed * 100) / memoryTotal) : 0,
    diskUsedGb: bytesToGb(diskUsed),
    diskTotalGb: bytesToGb(diskTotal),
    diskPercent: diskTotal > 0 ? Math.round((diskUsed * 100) / diskTotal) : undefined,
    uptimeSeconds: uptimeSeconds || undefined,
    latencyMs: latencyMs !== undefined && latencyMs > 0 ? latencyMs : undefined,
    lossPercent: lossPercent !== undefined && lossPercent >= 0 ? lossPercent : undefined,
    uplinkMbps: Math.round((txRate * 8) / 1_000_000),
    downlinkMbps: Math.round((rxRate * 8) / 1_000_000),
    usedTrafficGb: Math.round((rxBytes + txBytes) / 1024 / 1024 / 1024),
    quotaTrafficGb: Math.round(numberValue(agent.trafficLimit) / 1024 / 1024 / 1024),
    queue: numberValue(agent.queue),
    updatedAt: lastSeenAt,
    authStatus: agent.authStatus,
    lastHeartbeat: relativeTime(lastSeenAt),
    lastHeartbeatAt: lastSeenAt,
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

function firstNumber(...values: unknown[]): number {
  for (const value of values) {
    const parsed = numberValue(value);
    if (Number.isFinite(parsed) && parsed > 0) {
      return parsed;
    }
  }
  return 0;
}

function optionalNumber(value: unknown): number | undefined {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : undefined;
  }
  return undefined;
}

function bytesToGb(value: number): number | undefined {
  if (!Number.isFinite(value) || value <= 0) {
    return undefined;
  }
  return Math.round(value / 1024 / 1024 / 1024);
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

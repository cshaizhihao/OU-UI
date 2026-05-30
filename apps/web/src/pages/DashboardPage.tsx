import {
  AgentCards,
  AgentTable,
  RuntimeApplyPipeline,
  TaskStatePill
} from "../components/AgentViews";
import { AnalyticsPanel } from "../components/Charts";
import {
  getDeployRuntimeApply,
  getDeployTaskState,
  getRuntimeLabel,
  getTaskProgress,
  runtimeApplyStageLabel,
  runtimeApplyStages
} from "../controlFields";
import {
  agents,
  nodeHealthRows,
  protocolOptions,
  runtimeOptions,
  taskQueue,
  type Agent
} from "../data";
import {
  applyRouting,
  askCopilot,
  createAPIKey,
  createClashProfile,
  createLoadBalancer,
  createPanelUser,
  createRoutingRule,
  createSubscription,
  createTenant,
  createWebhook,
  importSubscription,
  optimizeAgent,
  type AlertEvent,
  type ClashProfile,
  type ControlTask,
  type DashboardDTO
} from "../api";
import { useMemo, useState, type FormEvent } from "react";

type DashboardPageProps = {
  data: DashboardDTO | null;
  loading?: boolean;
  error?: string;
  onRefresh?: () => void;
};

export function DashboardPage({ data, loading = false, error = "", onRefresh }: DashboardPageProps) {
  const liveAgents = data?.agents.length ? data.agents : agents;
  const onlineAgents = data?.overview.agentsOnline ?? liveAgents.filter((agent) => agent.status === "online").length;
  const totalUplink = liveAgents.reduce((sum, agent) => sum + agent.uplinkMbps, 0);
  const totalDownlink = liveAgents.reduce((sum, agent) => sum + agent.downlinkMbps, 0);
  const usedTraffic = liveAgents.reduce((sum, agent) => sum + agent.usedTrafficGb, 0);
  const quotaTraffic = liveAgents.reduce((sum, agent) => sum + agent.quotaTrafficGb, 0);
  const avgCPU = liveAgents.length
    ? Math.round(liveAgents.reduce((sum, agent) => sum + agent.cpu, 0) / liveAgents.length)
    : 0;
  const avgMemory = liveAgents.length
    ? Math.round(liveAgents.reduce((sum, agent) => sum + agent.memory, 0) / liveAgents.length)
    : 0;
  const kpis = [
    { label: "Online agents", value: `${onlineAgents} / ${data?.overview.agentsTotal ?? liveAgents.length}`, delta: data?.overview.version ?? "Fixture fallback" },
    { label: "Avg CPU", value: `${avgCPU}%`, delta: "Live heartbeat" },
    { label: "Avg memory", value: `${avgMemory}%`, delta: "Runtime sample" },
    { label: "Up / Down total", value: `${totalUplink} / ${totalDownlink} Mbps`, delta: "Live sample" },
    { label: "Traffic used", value: `${usedTraffic} GB`, delta: `Quota ${quotaTraffic} GB` }
  ];

  return (
    <div className="dashboard">
      {error ? (
        <section className="notice-row">
          <strong>{error}</strong>
          <button className="ghost-button" onClick={onRefresh} type="button">
            Retry
          </button>
        </section>
      ) : null}
      {loading ? <DashboardSkeleton /> : null}
      <section className="kpi-grid" id="overview">
        {kpis.map((kpi) => (
          <article className="kpi-card" key={kpi.label}>
            <span>{kpi.label}</span>
            <strong>{kpi.value}</strong>
            <em>{kpi.delta}</em>
          </article>
        ))}
      </section>

      <AgentCards agents={liveAgents} />

      <div className="split-grid">
        <section className="panel" id="deploy">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Node Dispatch</p>
              <h2>Runtime service control</h2>
            </div>
            <button className="primary-button">Queue control</button>
          </div>
          <form className="dispatch-form">
            <label>
              Agent
              <select defaultValue={liveAgents[0]?.id}>
                {liveAgents.map((agent) => (
                  <option value={agent.id} key={agent.id}>
                    {agent.name} - {agent.region}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Runtime
              <select defaultValue="Xray">
                {runtimeOptions.map((runtime) => (
                  <option value={runtime} key={runtime}>
                    {runtime}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Protocol
              <select defaultValue="VLESS Reality">
                {protocolOptions.map((protocol) => (
                  <option value={protocol} key={protocol}>
                    {protocol}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Task queue
              <select defaultValue="rolling">
                <option value="rolling">Managed reload - keep active sessions</option>
                <option value="immediate">Managed restart - maintenance window</option>
                <option value="staged">External service - approval required</option>
              </select>
            </label>
          </form>
          <div className="protocol-strip" aria-label="Available protocols">
            {protocolOptions.map((protocol) => (
              <span key={protocol}>{protocol}</span>
            ))}
          </div>
          <div className="protocol-strip" aria-label="Runtime service control stages">
            {runtimeApplyStages.map((stage) => (
              <span key={stage}>{runtimeApplyStageLabel[stage]}</span>
            ))}
          </div>
        </section>

        <section className="panel" id="queue">
          <div className="section-heading compact">
            <h2>Runtime service queue</h2>
            <button className="ghost-button">Pause queue</button>
          </div>
          <div className="task-list">
            {taskQueue.map((task) => {
              const taskState = getDeployTaskState(task);
              const runtimeApply = getDeployRuntimeApply(task);
              const progress = getTaskProgress(task);

              return (
                <article className="task-item" key={task.id}>
                  <div className="task-item-head">
                    <div>
                      <strong>{task.action}</strong>
                      <span>
                        {task.agentName} - {getRuntimeLabel(task.runtime)} - {task.protocol}
                      </span>
                    </div>
                    <small>{task.eta ?? taskState.status}</small>
                  </div>
                  <div className="progress">
                    <span
                      className={`progress-${taskState.status}`}
                      style={{ width: `${progress}%` }}
                    />
                  </div>
                  <div className="task-meta">
                    <span>{task.id}</span>
                    <TaskStatePill status={taskState.status} />
                    <span>{runtimeApply.runtimeVersion}</span>
                    <span>{formatServiceStatus(runtimeApply.serviceStatus)}</span>
                    <span>{formatServiceStatus(runtimeApply.serviceMode)} mode</span>
                    <span>{runtimeApply.runtimeManaged ? "OU-UI managed" : "External service"}</span>
                    <span>
                      {runtimeApply.rollbackAvailable
                        ? "Rollback available"
                        : "Rollback unavailable"}
                    </span>
                    <span>Retries {taskState.retryCount}</span>
                  </div>
                  <div className="task-runtime-detail">
                    <div>
                      <span>Control stage</span>
                      <strong>{runtimeApplyStageLabel[runtimeApply.currentStage]}</strong>
                    </div>
                    <div>
                      <span>Unit path</span>
                      <strong title={runtimeApply.unitPath}>{runtimeApply.unitPath}</strong>
                    </div>
                    <div>
                      <span>Config dir</span>
                      <strong title={runtimeApply.configDir}>{runtimeApply.configDir}</strong>
                    </div>
                    <div>
                      <span>Config path</span>
                      <strong title={runtimeApply.configPath}>{runtimeApply.configPath}</strong>
                    </div>
                    <div>
                      <span>Reload</span>
                      <strong title={runtimeApply.reloadInfo}>
                        {formatServiceStatus(runtimeApply.reloadStatus)}
                      </strong>
                    </div>
                    <div>
                      <span>Restart</span>
                      <strong title={runtimeApply.restartInfo}>
                        {formatServiceStatus(runtimeApply.restartStatus)}
                      </strong>
                    </div>
                    <div>
                      <span>Health</span>
                      <strong title={runtimeApply.healthInfo}>
                        {formatServiceStatus(runtimeApply.healthStatus)}
                      </strong>
                    </div>
                  </div>
                  <RuntimeApplyPipeline apply={runtimeApply} />
                  {runtimeApply.failureStage ? (
                    <p className="task-reason">
                      Failed stage: {runtimeApplyStageLabel[runtimeApply.failureStage]}
                    </p>
                  ) : null}
                  {taskState.failureReason ? (
                    <p className="task-reason">
                      Failure reason: {taskState.failureReason}
                    </p>
                  ) : null}
                </article>
              );
            })}
          </div>
        </section>
      </div>

      <AnalyticsPanel />

      <V3ControlCenter data={data} agents={liveAgents} onRefresh={onRefresh} />

      <section className="panel" id="nodes">
        <div className="section-heading compact">
          <h2>Node health</h2>
          <button className="ghost-button">Export snapshot</button>
        </div>
        <div className="node-list">
          {nodeHealthRows.map((node) => (
            <article className="node-item" key={node.name}>
              <div>
                <strong>{node.name}</strong>
                <span>{node.detail}</span>
              </div>
              <small>{node.value}</small>
            </article>
          ))}
        </div>
      </section>

      <AgentTable agents={liveAgents} />
    </div>
  );
}

function formatServiceStatus(status: string): string {
  return status
    .replace(/[_-]+/g, " ")
    .replace(/\s+/g, " ")
    .trim()
    .replace(/\b[a-z]/g, (letter) => letter.toUpperCase());
}

function DashboardSkeleton() {
  return (
    <section className="skeleton-grid" aria-label="Dashboard loading">
      {Array.from({ length: 5 }).map((_, index) => (
        <span key={index} />
      ))}
    </section>
  );
}

function V3ControlCenter({
  data,
  agents,
  onRefresh
}: {
  data: DashboardDTO | null;
  agents: Agent[];
  onRefresh?: () => void;
}) {
  const control = data?.control;
  const [busy, setBusy] = useState("");
  const [message, setMessage] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [routing, setRouting] = useState({
    name: "Block ads",
    ruleType: "ads",
    match: "category-ads-all",
    action: "block",
    priority: 10
  });
  const [loadBalancer, setLoadBalancer] = useState({
    name: "Global HA",
    entryTag: "ou-ha",
    strategy: "latency-loss",
    healthCheckInterval: 30
  });
  const [webhook, setWebhook] = useState({
    name: "Ops hook",
    kind: "generic",
    url: "",
    secret: "",
    chatId: "",
    eventTypes: "agent.offline,traffic.quota.exceeded"
  });
  const [subscription, setSubscription] = useState({
    name: "External pool",
    url: "",
    content: ""
  });
  const [clash, setClash] = useState({
    name: "OU-UI Managed Clash",
    providerUrl: "https://example.com/rules/private.yaml"
  });
  const [tenant, setTenant] = useState({
    name: "Ops tenant",
    nodeAccess: agents[0]?.id ?? "",
    monthlyTrafficGb: 1024,
    maxConnections: 2000
  });
  const [panelUser, setPanelUser] = useState({
    username: "operator",
    password: "change-me-now",
    tenantId: "",
    nodeAccess: agents[0]?.id ?? "",
    monthlyTrafficGb: 256,
    maxConnections: 500
  });
  const [keyForm, setKeyForm] = useState({
    name: "Billing integration",
    tenantId: "",
    scopes: "panel:read"
  });
  const [question, setQuestion] = useState("Why is the newest Agent degraded?");

  const trafficTotal = useMemo(() => {
    const rx = control?.traffic.reduce((sum, item) => sum + item.rxBytes, 0) ?? 0;
    const tx = control?.traffic.reduce((sum, item) => sum + item.txBytes, 0) ?? 0;
    const connections = control?.traffic.reduce((sum, item) => sum + item.connections, 0) ?? 0;
    return { bytes: rx + tx, connections };
  }, [control?.traffic]);

  async function runAction(label: string, action: () => Promise<unknown>) {
    setBusy(label);
    setMessage("");
    setApiKey("");
    try {
      const result = await action();
      if (isAPIKeyResponse(result)) {
        setApiKey(result.apiKey);
      }
      setMessage(`${label} completed`);
      await Promise.resolve(onRefresh?.());
    } catch (err) {
      setMessage(err instanceof Error ? err.message : `${label} failed`);
    } finally {
      setBusy("");
    }
  }

  function handleCreateRouting(event: FormEvent) {
    event.preventDefault();
    void runAction("Routing rule", () =>
      createRoutingRule({
        ...routing,
        enabled: true,
        protocol: "",
        targetTag: routing.action === "proxy" ? "OU-Auto" : "",
        description: "Managed from OU-UI V3 console"
      })
    );
  }

  function handleApplyRouting() {
    const agentIds = agents.filter((agent) => agent.status !== "offline").map((agent) => agent.id);
    void runAction("Routing apply", () => applyRouting(agentIds));
  }

  function handleOptimize() {
    const agent = agents.find((item) => item.status !== "offline") ?? agents[0];
    if (!agent) {
      setMessage("No Agent is available for host tuning");
      return;
    }
    void runAction("Host tuning", () => optimizeAgent(agent.id));
  }

  function handleCreateLoadBalancer(event: FormEvent) {
    event.preventDefault();
    const members = agents.slice(0, 4).map((agent, index) => ({
      id: agent.id,
      name: agent.name,
      latencyMs: 40 + index * 15,
      lossPercent: index === 0 ? 0 : 0.2,
      weight: 1
    }));
    void runAction("HA group", () =>
      createLoadBalancer({
        ...loadBalancer,
        members,
        healthCheckInterval: Number(loadBalancer.healthCheckInterval) || 30
      })
    );
  }

  function handleCreateWebhook(event: FormEvent) {
    event.preventDefault();
    void runAction("Webhook", () =>
      createWebhook({
        ...webhook,
        enabled: true,
        eventTypes: parseCSV(webhook.eventTypes)
      })
    );
  }

  function handleCreateSubscription(event: FormEvent) {
    event.preventDefault();
    void runAction("Subscription", () =>
      createSubscription({
        name: subscription.name,
        url: subscription.url,
        content: subscription.content,
        format: "auto",
        enabled: true
      })
    );
  }

  function handleImportFirstSubscription() {
    const sub = control?.subscriptions[0];
    if (!sub) {
      setMessage("Create a subscription first");
      return;
    }
    void runAction("Subscription import", () => importSubscription(sub.id));
  }

  function handleCreateClash(event: FormEvent) {
    event.preventDefault();
    void runAction("Clash profile", () =>
      createClashProfile({
        name: clash.name,
        ruleProviders: clash.providerUrl
          ? [
              {
                name: "private",
                type: "http",
                behavior: "domain",
                url: clash.providerUrl,
                interval: 86400
              }
            ]
          : [],
        proxyGroups: [],
        routingRules: []
      })
    );
  }

  function handleCreateTenant(event: FormEvent) {
    event.preventDefault();
    void runAction("Tenant", () =>
      createTenant({
        name: tenant.name,
        status: "active",
        role: "operator",
        nodeAccess: parseCSV(tenant.nodeAccess),
        monthlyTrafficQuota: gbToBytes(tenant.monthlyTrafficGb),
        maxConnections: Number(tenant.maxConnections) || 0
      })
    );
  }

  function handleCreatePanelUser(event: FormEvent) {
    event.preventDefault();
    void runAction("Panel user", () =>
      createPanelUser({
        username: panelUser.username,
        password: panelUser.password,
        tenantId: panelUser.tenantId,
        role: "operator",
        status: "active",
        nodeAccess: parseCSV(panelUser.nodeAccess),
        monthlyTrafficQuota: gbToBytes(panelUser.monthlyTrafficGb),
        maxConnections: Number(panelUser.maxConnections) || 0
      })
    );
  }

  function handleCreateAPIKey(event: FormEvent) {
    event.preventDefault();
    void runAction("API key", () =>
      createAPIKey({
        name: keyForm.name,
        tenantId: keyForm.tenantId,
        scopes: parseCSV(keyForm.scopes),
        status: "active"
      })
    );
  }

  function handleAskCopilot(event: FormEvent) {
    event.preventDefault();
    void runAction("Copilot", () => askCopilot(question));
  }

  return (
    <section className="v3-grid" aria-label="V3 control center">
      <div className="control-banner">
        <div>
          <p className="eyebrow">V3 Control Center</p>
          <h2>Traffic, routing, HA, subscriptions, RBAC, API, and Copilot</h2>
        </div>
        <div className="control-banner-metrics">
          <MetricBox label="Managed nodes" value={String(control?.nodes.length ?? 0)} />
          <MetricBox label="Per-node traffic" value={formatBytes(trafficTotal.bytes)} />
          <MetricBox label="Connections" value={String(trafficTotal.connections)} />
          <MetricBox label="Open alerts" value={String(control?.alerts.length ?? 0)} />
        </div>
      </div>

      {message ? (
        <div className="notice-row compact">
          <strong>{message}</strong>
          {apiKey ? <code>{apiKey}</code> : null}
        </div>
      ) : null}

      <div className="control-two">
        <section className="panel control-panel" id="routing">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Routing</p>
              <h2>Geo rules and host tuning</h2>
            </div>
            <div className="button-row">
              <button className="ghost-button" disabled={Boolean(busy)} onClick={handleOptimize} type="button">
                BBR v3
              </button>
              <button className="primary-button" disabled={Boolean(busy)} onClick={handleApplyRouting} type="button">
                Apply
              </button>
            </div>
          </div>
          <form className="control-form" onSubmit={handleCreateRouting}>
            <label>
              Name
              <input value={routing.name} onChange={(event) => setRouting({ ...routing, name: event.target.value })} />
            </label>
            <label>
              Type
              <select value={routing.ruleType} onChange={(event) => setRouting({ ...routing, ruleType: event.target.value })}>
                <option value="geoip">GeoIP</option>
                <option value="geosite">GeoSite</option>
                <option value="ads">Ads</option>
                <option value="domain">Domain</option>
                <option value="protocol">Protocol</option>
                <option value="ip">IP CIDR</option>
              </select>
            </label>
            <label>
              Match
              <input value={routing.match} onChange={(event) => setRouting({ ...routing, match: event.target.value })} />
            </label>
            <label>
              Action
              <select value={routing.action} onChange={(event) => setRouting({ ...routing, action: event.target.value })}>
                <option value="block">Block</option>
                <option value="direct">Direct</option>
                <option value="proxy">Proxy</option>
              </select>
            </label>
            <button className="primary-button" disabled={Boolean(busy)} type="submit">
              Save rule
            </button>
          </form>
          <MiniTable
            columns={["Rule", "Match", "Action"]}
            rows={(control?.routingRules ?? []).slice(0, 6).map((rule) => [rule.name, `${rule.ruleType}:${rule.match}`, rule.action])}
          />
        </section>

        <section className="panel control-panel" id="ha">
          <div className="section-heading">
            <div>
              <p className="eyebrow">High Availability</p>
              <h2>Latency-loss balancing</h2>
            </div>
          </div>
          <form className="control-form" onSubmit={handleCreateLoadBalancer}>
            <label>
              Group
              <input value={loadBalancer.name} onChange={(event) => setLoadBalancer({ ...loadBalancer, name: event.target.value })} />
            </label>
            <label>
              Entry tag
              <input value={loadBalancer.entryTag} onChange={(event) => setLoadBalancer({ ...loadBalancer, entryTag: event.target.value })} />
            </label>
            <label>
              Strategy
              <select value={loadBalancer.strategy} onChange={(event) => setLoadBalancer({ ...loadBalancer, strategy: event.target.value })}>
                <option value="latency-loss">Latency and loss</option>
                <option value="weighted">Weighted</option>
              </select>
            </label>
            <label>
              Check seconds
              <input
                type="number"
                value={loadBalancer.healthCheckInterval}
                onChange={(event) => setLoadBalancer({ ...loadBalancer, healthCheckInterval: Number(event.target.value) })}
              />
            </label>
            <button className="primary-button" disabled={Boolean(busy)} type="submit">
              Create group
            </button>
          </form>
          <MiniTable
            columns={["Group", "Selected", "Status"]}
            rows={(control?.loadBalancers ?? []).slice(0, 6).map((group) => [group.name, String(group.lastDecision?.selected ?? "-"), group.status])}
          />
        </section>
      </div>

      <div className="control-two">
        <section className="panel control-panel" id="alerts">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Webhooks</p>
              <h2>Alert delivery</h2>
            </div>
          </div>
          <form className="control-form" onSubmit={handleCreateWebhook}>
            <label>
              Name
              <input value={webhook.name} onChange={(event) => setWebhook({ ...webhook, name: event.target.value })} />
            </label>
            <label>
              Kind
              <select value={webhook.kind} onChange={(event) => setWebhook({ ...webhook, kind: event.target.value })}>
                <option value="generic">Generic</option>
                <option value="telegram">Telegram</option>
                <option value="serverchan">ServerChan</option>
              </select>
            </label>
            <label>
              URL
              <input value={webhook.url} onChange={(event) => setWebhook({ ...webhook, url: event.target.value })} />
            </label>
            <label>
              Events
              <input value={webhook.eventTypes} onChange={(event) => setWebhook({ ...webhook, eventTypes: event.target.value })} />
            </label>
            <button className="primary-button" disabled={Boolean(busy)} type="submit">
              Save hook
            </button>
          </form>
          <AlertList alerts={control?.alerts ?? []} />
        </section>

        <section className="panel control-panel" id="subscriptions">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Subscriptions</p>
              <h2>External node aggregation</h2>
            </div>
            <button className="ghost-button" disabled={Boolean(busy)} onClick={handleImportFirstSubscription} type="button">
              Import first
            </button>
          </div>
          <form className="control-form" onSubmit={handleCreateSubscription}>
            <label>
              Name
              <input value={subscription.name} onChange={(event) => setSubscription({ ...subscription, name: event.target.value })} />
            </label>
            <label>
              URL
              <input value={subscription.url} onChange={(event) => setSubscription({ ...subscription, url: event.target.value })} />
            </label>
            <label className="full-span">
              Inline content
              <textarea value={subscription.content} onChange={(event) => setSubscription({ ...subscription, content: event.target.value })} />
            </label>
            <button className="primary-button" disabled={Boolean(busy)} type="submit">
              Add source
            </button>
          </form>
          <MiniTable
            columns={["Node", "Protocol", "Endpoint"]}
            rows={(control?.externalNodes ?? []).slice(0, 6).map((node) => [node.name, node.protocol, `${node.address}:${node.port}`])}
          />
        </section>
      </div>

      <div className="control-two">
        <section className="panel control-panel" id="clash">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Clash</p>
              <h2>Profile hosting</h2>
            </div>
          </div>
          <form className="control-form" onSubmit={handleCreateClash}>
            <label>
              Name
              <input value={clash.name} onChange={(event) => setClash({ ...clash, name: event.target.value })} />
            </label>
            <label>
              Provider URL
              <input value={clash.providerUrl} onChange={(event) => setClash({ ...clash, providerUrl: event.target.value })} />
            </label>
            <button className="primary-button" disabled={Boolean(busy)} type="submit">
              Generate YAML
            </button>
          </form>
          <ProfileList profiles={control?.clashProfiles ?? []} />
        </section>

        <section className="panel control-panel" id="rbac">
          <div className="section-heading">
            <div>
              <p className="eyebrow">RBAC</p>
              <h2>Tenants and sub-users</h2>
            </div>
          </div>
          <div className="nested-forms">
            <form className="control-form" onSubmit={handleCreateTenant}>
              <label>
                Tenant
                <input value={tenant.name} onChange={(event) => setTenant({ ...tenant, name: event.target.value })} />
              </label>
              <label>
                Node access
                <input value={tenant.nodeAccess} onChange={(event) => setTenant({ ...tenant, nodeAccess: event.target.value })} />
              </label>
              <label>
                GB quota
                <input
                  type="number"
                  value={tenant.monthlyTrafficGb}
                  onChange={(event) => setTenant({ ...tenant, monthlyTrafficGb: Number(event.target.value) })}
                />
              </label>
              <label>
                Connections
                <input
                  type="number"
                  value={tenant.maxConnections}
                  onChange={(event) => setTenant({ ...tenant, maxConnections: Number(event.target.value) })}
                />
              </label>
              <button className="primary-button" disabled={Boolean(busy)} type="submit">
                Create tenant
              </button>
            </form>
            <form className="control-form" onSubmit={handleCreatePanelUser}>
              <label>
                User
                <input value={panelUser.username} onChange={(event) => setPanelUser({ ...panelUser, username: event.target.value })} />
              </label>
              <label>
                Password
                <input value={panelUser.password} onChange={(event) => setPanelUser({ ...panelUser, password: event.target.value })} />
              </label>
              <label>
                Tenant ID
                <input value={panelUser.tenantId} onChange={(event) => setPanelUser({ ...panelUser, tenantId: event.target.value })} />
              </label>
              <label>
                Node access
                <input value={panelUser.nodeAccess} onChange={(event) => setPanelUser({ ...panelUser, nodeAccess: event.target.value })} />
              </label>
              <button className="ghost-button" disabled={Boolean(busy)} type="submit">
                Create user
              </button>
            </form>
          </div>
          <MiniTable
            columns={["Tenant", "Role", "Quota"]}
            rows={(control?.tenants ?? []).slice(0, 4).map((item) => [item.name, item.role, formatBytes(item.monthlyTrafficQuota ?? 0)])}
          />
        </section>
      </div>

      <div className="control-two">
        <section className="panel control-panel" id="api">
          <div className="section-heading">
            <div>
              <p className="eyebrow">REST API</p>
              <h2>Scoped integration keys</h2>
            </div>
          </div>
          <form className="control-form" onSubmit={handleCreateAPIKey}>
            <label>
              Name
              <input value={keyForm.name} onChange={(event) => setKeyForm({ ...keyForm, name: event.target.value })} />
            </label>
            <label>
              Tenant ID
              <input value={keyForm.tenantId} onChange={(event) => setKeyForm({ ...keyForm, tenantId: event.target.value })} />
            </label>
            <label>
              Scopes
              <input value={keyForm.scopes} onChange={(event) => setKeyForm({ ...keyForm, scopes: event.target.value })} />
            </label>
            <button className="primary-button" disabled={Boolean(busy)} type="submit">
              Issue key
            </button>
          </form>
          <TaskRail tasks={control?.tasks ?? []} />
        </section>

        <section className="panel control-panel" id="copilot">
          <div className="section-heading">
            <div>
              <p className="eyebrow">AI Copilot</p>
              <h2>Incident diagnosis</h2>
            </div>
          </div>
          <form className="control-form" onSubmit={handleAskCopilot}>
            <label className="full-span">
              Question
              <textarea value={question} onChange={(event) => setQuestion(event.target.value)} />
            </label>
            <button className="primary-button" disabled={Boolean(busy)} type="submit">
              Ask
            </button>
          </form>
          <div className="incident-list">
            {(control?.copilotIncidents ?? []).slice(0, 3).map((incident) => (
              <article key={incident.id}>
                <strong>{incident.question}</strong>
                <span>
                  {incident.model} / {incident.status} / {formatTime(incident.createdAt)}
                </span>
                <p>{incident.answer}</p>
              </article>
            ))}
          </div>
        </section>
      </div>
    </section>
  );
}

function MetricBox({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function MiniTable({ columns, rows }: { columns: string[]; rows: string[][] }) {
  return (
    <div className="mini-table">
      <div className="mini-row mini-head">
        {columns.map((column) => (
          <strong key={column}>{column}</strong>
        ))}
      </div>
      {rows.length ? (
        rows.map((row, index) => (
          <div className="mini-row" key={`${row.join("-")}-${index}`}>
            {row.map((cell, cellIndex) => (
              <span key={`${cell}-${cellIndex}`}>{cell || "-"}</span>
            ))}
          </div>
        ))
      ) : (
        <div className="mini-row empty">
          <span>No records yet</span>
        </div>
      )}
    </div>
  );
}

function AlertList({ alerts }: { alerts: AlertEvent[] }) {
  return (
    <div className="alert-list">
      {alerts.slice(0, 5).map((alert) => (
        <article className={`alert-item alert-${alert.severity}`} key={alert.id}>
          <div>
            <strong>{alert.eventType}</strong>
            <span>
              {alert.sourceType}:{alert.sourceId} / {formatTime(alert.createdAt)}
            </span>
          </div>
          <small>{alert.delivered ? "delivered" : alert.lastError || "pending"}</small>
        </article>
      ))}
      {alerts.length === 0 ? <p className="empty-state">No alerts yet</p> : null}
    </div>
  );
}

function ProfileList({ profiles }: { profiles: ClashProfile[] }) {
  return (
    <div className="profile-list">
      {profiles.slice(0, 5).map((profile) => (
        <article key={profile.id}>
          <strong>{profile.name}</strong>
          <code>{profileYAMLPath(profile.id)}</code>
          <span>{formatTime(profile.updatedAt)}</span>
        </article>
      ))}
      {profiles.length === 0 ? <p className="empty-state">No profiles yet</p> : null}
    </div>
  );
}

function profileYAMLPath(id: string): string {
  return `/api/v1/clash/profiles/${id}.yaml`;
}

function TaskRail({ tasks }: { tasks: ControlTask[] }) {
  return (
    <div className="task-rail">
      {tasks.slice(0, 8).map((task) => (
        <span className={`task-state task-state-${taskTone(task.status)}`} key={task.id} title={task.lastError || task.type}>
          {task.type} / {task.status}
        </span>
      ))}
      {tasks.length === 0 ? <p className="empty-state">No tasks yet</p> : null}
    </div>
  );
}

function parseCSV(value: string): string[] {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function gbToBytes(value: number): number {
  return Math.max(0, Math.round(Number(value) || 0)) * 1024 * 1024 * 1024;
}

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) {
    return "0 GB";
  }
  if (value >= 1024 * 1024 * 1024) {
    return `${Math.round(value / 1024 / 1024 / 1024)} GB`;
  }
  return `${Math.round(value / 1024 / 1024)} MB`;
}

function formatTime(value?: string): string {
  if (!value) {
    return "never";
  }
  const time = new Date(value).getTime();
  if (!Number.isFinite(time)) {
    return value;
  }
  const minutes = Math.max(0, Math.round((Date.now() - time) / 60000));
  if (minutes < 60) {
    return `${minutes}m ago`;
  }
  const hours = Math.round(minutes / 60);
  return hours < 48 ? `${hours}h ago` : `${Math.round(hours / 24)}d ago`;
}

function taskTone(status: string): "pending" | "running" | "success" | "failed" {
  if (status === "succeeded" || status === "success") {
    return "success";
  }
  if (status === "running") {
    return "running";
  }
  if (status === "failed" || status === "canceled") {
    return "failed";
  }
  return "pending";
}

function isAPIKeyResponse(value: unknown): value is { apiKey: string } {
  return Boolean(value && typeof value === "object" && "apiKey" in value);
}

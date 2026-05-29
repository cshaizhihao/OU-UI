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
  taskQueue
} from "../data";
import type { DashboardDTO } from "../api";

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

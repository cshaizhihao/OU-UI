import { AgentCards, AgentTable, TaskStatePill } from "../components/AgentViews";
import { AnalyticsPanel } from "../components/Charts";
import { getDeployTaskState, getRuntimeLabel, getTaskProgress } from "../controlFields";
import {
  agents,
  nodeHealthRows,
  protocolOptions,
  runtimeOptions,
  taskQueue
} from "../data";

const onlineAgents = agents.filter((agent) => agent.status === "online").length;
const totalUplink = agents.reduce((sum, agent) => sum + agent.uplinkMbps, 0);
const totalDownlink = agents.reduce((sum, agent) => sum + agent.downlinkMbps, 0);
const usedTraffic = agents.reduce((sum, agent) => sum + agent.usedTrafficGb, 0);
const quotaTraffic = agents.reduce((sum, agent) => sum + agent.quotaTrafficGb, 0);

const kpis = [
  { label: "Online agents", value: `${onlineAgents} / ${agents.length}`, delta: "1 degraded" },
  { label: "Avg CPU", value: "40%", delta: "Peak 76%" },
  { label: "Avg memory", value: "49%", delta: "Stable" },
  { label: "Up / Down total", value: `${totalUplink} / ${totalDownlink} Mbps`, delta: "Live sample" },
  { label: "Traffic used", value: `${usedTraffic} GB`, delta: `Quota ${quotaTraffic} GB` }
];

export function DashboardPage() {
  return (
    <div className="dashboard">
      <section className="kpi-grid" id="overview">
        {kpis.map((kpi) => (
          <article className="kpi-card" key={kpi.label}>
            <span>{kpi.label}</span>
            <strong>{kpi.value}</strong>
            <em>{kpi.delta}</em>
          </article>
        ))}
      </section>

      <AgentCards agents={agents} />

      <div className="split-grid">
        <section className="panel" id="deploy">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Node Dispatch</p>
              <h2>Node delivery</h2>
            </div>
            <button className="primary-button">Enqueue</button>
          </div>
          <form className="dispatch-form">
            <label>
              Agent
              <select defaultValue={agents[0].id}>
                {agents.map((agent) => (
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
                <option value="rolling">Rolling delivery - keep active sessions</option>
                <option value="immediate">Immediate delivery - reload runtime</option>
                <option value="staged">Staged delivery - approval required</option>
              </select>
            </label>
          </form>
          <div className="protocol-strip" aria-label="Available protocols">
            {protocolOptions.map((protocol) => (
              <span key={protocol}>{protocol}</span>
            ))}
          </div>
        </section>

        <section className="panel" id="queue">
          <div className="section-heading compact">
            <h2>Task queue</h2>
            <button className="ghost-button">Pause queue</button>
          </div>
          <div className="task-list">
            {taskQueue.map((task) => {
              const taskState = getDeployTaskState(task);
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
                    <span>Retries {taskState.retryCount}</span>
                  </div>
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

      <AgentTable agents={agents} />
    </div>
  );
}

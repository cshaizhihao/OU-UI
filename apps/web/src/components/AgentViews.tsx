import type { Agent, AgentStatus } from "../data";

const statusLabel: Record<AgentStatus, string> = {
  online: "在线",
  busy: "繁忙",
  idle: "空闲",
  offline: "离线"
};

type AgentViewsProps = {
  agents: Agent[];
};

export function AgentCards({ agents }: AgentViewsProps) {
  return (
    <section className="panel" id="Agents">
      <div className="section-heading">
        <div>
          <p className="eyebrow">Agent Fleet</p>
          <h2>活跃 Agent</h2>
        </div>
        <button className="ghost-button">管理编排</button>
      </div>
      <div className="agent-grid">
        {agents.map((agent) => (
          <article className="agent-card" key={agent.id}>
            <div className="agent-card-head">
              <div>
                <h3>{agent.name}</h3>
                <span>{agent.role}</span>
              </div>
              <StatusPill status={agent.status} />
            </div>
            <div className="agent-metrics">
              <Metric label="队列" value={String(agent.queue)} />
              <Metric label="成功率" value={`${agent.successRate}%`} />
              <Metric label="延迟" value={agent.latency} />
            </div>
          </article>
        ))}
      </div>
    </section>
  );
}

export function AgentTable({ agents }: AgentViewsProps) {
  return (
    <section className="panel">
      <div className="section-heading compact">
        <h2>Agent 明细</h2>
        <div className="segmented">
          <button className="selected">全部</button>
          <button>在线</button>
          <button>异常</button>
        </div>
      </div>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Agent</th>
              <th>状态</th>
              <th>队列</th>
              <th>成功率</th>
              <th>成本</th>
              <th>更新</th>
            </tr>
          </thead>
          <tbody>
            {agents.map((agent) => (
              <tr key={agent.id}>
                <td>
                  <strong>{agent.name}</strong>
                  <span>{agent.id}</span>
                </td>
                <td>
                  <StatusPill status={agent.status} />
                </td>
                <td>{agent.queue}</td>
                <td>{agent.successRate}%</td>
                <td>{agent.cost}</td>
                <td>{agent.updatedAt}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function StatusPill({ status }: { status: AgentStatus }) {
  return <span className={`status status-${status}`}>{statusLabel[status]}</span>;
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

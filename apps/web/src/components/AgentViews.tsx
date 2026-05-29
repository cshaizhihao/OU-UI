import type { Agent, AgentStatus } from "../data";

const statusLabel: Record<AgentStatus, string> = {
  online: "Online",
  degraded: "Degraded",
  offline: "Offline"
};

type AgentViewsProps = {
  agents: Agent[];
};

export function AgentCards({ agents }: AgentViewsProps) {
  return (
    <section className="panel" id="agents">
      <div className="section-heading">
        <div>
          <p className="eyebrow">Agent Monitor</p>
          <h2>Proxy node monitor</h2>
        </div>
        <button className="ghost-button">Sync probes</button>
      </div>
      <div className="agent-grid">
        {agents.map((agent) => (
          <article className="agent-card" key={agent.id}>
            <div className="agent-card-head">
              <div>
                <h3>{agent.name}</h3>
                <span>
                  {agent.region} - {agent.ip}
                </span>
              </div>
              <StatusPill status={agent.status} />
            </div>

            <div className="agent-card-meta">
              <span>{agent.runtime}</span>
              <span>Queue {agent.queue}</span>
              <span>{agent.updatedAt}</span>
            </div>

            <div className="agent-bars">
              <MeterLine label="CPU" value={agent.cpu} suffix="%" />
              <MeterLine label="Memory" value={agent.memory} suffix="%" />
            </div>

            <div className="agent-metrics">
              <Metric label="Uplink" value={`${agent.uplinkMbps} Mbps`} />
              <Metric label="Downlink" value={`${agent.downlinkMbps} Mbps`} />
              <Metric
                label="Traffic"
                value={`${agent.usedTrafficGb} / ${agent.quotaTrafficGb} GB`}
              />
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
        <h2>Agent detail</h2>
        <div className="segmented" aria-label="Agent status filter">
          <button className="selected">All</button>
          <button>Online</button>
          <button>Issues</button>
        </div>
      </div>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Agent</th>
              <th>Status</th>
              <th>Runtime</th>
              <th>CPU</th>
              <th>Memory</th>
              <th>Up / Down</th>
              <th>Traffic quota</th>
              <th>Updated</th>
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
                <td>{agent.runtime}</td>
                <td>{agent.cpu}%</td>
                <td>{agent.memory}%</td>
                <td>
                  {agent.uplinkMbps} / {agent.downlinkMbps} Mbps
                </td>
                <td>
                  {agent.usedTrafficGb} / {agent.quotaTrafficGb} GB
                </td>
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

function MeterLine({ label, value, suffix }: { label: string; value: number; suffix: string }) {
  return (
    <div className="meter-line">
      <div>
        <span>{label}</span>
        <strong>
          {value}
          {suffix}
        </strong>
      </div>
      <div className="meter-track">
        <span style={{ width: `${value}%` }} />
      </div>
    </div>
  );
}

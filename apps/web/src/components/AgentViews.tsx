import {
  getAgentRuntimeApply,
  getAgentTaskState,
  getAuthState,
  getLastHeartbeat,
  getRegistrationState,
  getRuntimeCapabilities,
  getRuntimeLabel,
  runtimeApplyStageLabel,
  type ControlBadgeState,
  type RuntimeApplyView
} from "../controlFields";
import type { Agent, AgentStatus, ControlTaskStatus, RuntimeApplyStage } from "../data";
import type { ReactNode } from "react";

const statusLabel: Record<AgentStatus, string> = {
  online: "Online",
  degraded: "Degraded",
  offline: "Offline"
};

const taskStatusLabel: Record<ControlTaskStatus, string> = {
  pending: "pending",
  running: "running",
  success: "success",
  failed: "failed"
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
        {agents.map((agent) => {
          const registration = getRegistrationState(agent);
          const auth = getAuthState(agent);
          const heartbeat = getLastHeartbeat(agent);
          const runtime = getRuntimeLabel(agent.runtime);
          const capabilities = getRuntimeCapabilities(agent);
          const task = getAgentTaskState(agent);
          const runtimeApply = getAgentRuntimeApply(agent);

          return (
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
                <span>{runtime}</span>
                <span>{runtimeApply.runtimeVersion}</span>
                <span>{formatServiceStatus(runtimeApply.serviceStatus)}</span>
                <span>Queue {agent.queue}</span>
              </div>

              <div className="control-summary" aria-label="Agent control chain">
                <ControlDatum label="Registration">
                  <ControlPill state={registration} />
                </ControlDatum>
                <ControlDatum label="Auth">
                  <ControlPill state={auth} />
                </ControlDatum>
                <ControlDatum label="Last heartbeat" value={heartbeat} />
              </div>

              <RuntimeApplySummary apply={runtimeApply} />

              <div className="agent-task-summary">
                <div>
                  <span>Task</span>
                  <TaskStatePill status={task.status} />
                </div>
                <div>
                  <span>Retries</span>
                  <strong>{task.retryCount}</strong>
                </div>
                {task.failureReason ? (
                  <p className="failure-reason">Failure reason: {task.failureReason}</p>
                ) : null}
              </div>

              <CapabilityList capabilities={capabilities} />

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
          );
        })}
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
              <th>Control</th>
              <th>Runtime</th>
              <th>Apply stages</th>
              <th>Config / rollback</th>
              <th>Capabilities</th>
              <th>CPU</th>
              <th>Memory</th>
              <th>Up / Down</th>
              <th>Traffic quota</th>
              <th>Heartbeat</th>
              <th>Task</th>
            </tr>
          </thead>
          <tbody>
            {agents.map((agent) => {
              const registration = getRegistrationState(agent);
              const auth = getAuthState(agent);
              const heartbeat = getLastHeartbeat(agent);
              const runtime = getRuntimeLabel(agent.runtime);
              const capabilities = getRuntimeCapabilities(agent);
              const task = getAgentTaskState(agent);
              const runtimeApply = getAgentRuntimeApply(agent);

              return (
                <tr key={agent.id}>
                  <td>
                    <strong>{agent.name}</strong>
                    <span>{agent.id}</span>
                  </td>
                  <td>
                    <StatusPill status={agent.status} />
                  </td>
                  <td>
                    <div className="control-stack">
                      <ControlPill state={registration} />
                      <ControlPill state={auth} />
                    </div>
                  </td>
                  <td>
                    <div className="runtime-cell">
                      <strong>{runtime}</strong>
                      <span>{runtimeApply.runtimeVersion}</span>
                      <ServiceStatusPill status={runtimeApply.serviceStatus} />
                    </div>
                  </td>
                  <td>
                    <RuntimeApplyPipeline apply={runtimeApply} compact />
                    <RuntimeFailureStage stage={runtimeApply.failureStage} />
                  </td>
                  <td>
                    <div className="config-cell">
                      <span title={runtimeApply.configPath}>{runtimeApply.configPath}</span>
                      <strong>
                        {runtimeApply.rollbackAvailable
                          ? "Rollback available"
                          : "Rollback unavailable"}
                      </strong>
                    </div>
                  </td>
                  <td>
                    <CapabilityList capabilities={capabilities} compact />
                  </td>
                  <td>{agent.cpu}%</td>
                  <td>{agent.memory}%</td>
                  <td>
                    {agent.uplinkMbps} / {agent.downlinkMbps} Mbps
                  </td>
                  <td>
                    {agent.usedTrafficGb} / {agent.quotaTrafficGb} GB
                  </td>
                  <td>{heartbeat}</td>
                  <td>
                    <div className="task-cell">
                      <TaskStatePill status={task.status} />
                      <span>Retries {task.retryCount}</span>
                      {task.failureReason ? <em>{task.failureReason}</em> : null}
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function StatusPill({ status }: { status: AgentStatus }) {
  return <span className={`status status-${status}`}>{statusLabel[status]}</span>;
}

function ControlPill({ state }: { state: ControlBadgeState }) {
  return <span className={`status control-pill control-${state.tone}`}>{state.label}</span>;
}

function ServiceStatusPill({ status }: { status: string }) {
  return (
    <span className={`status service-pill service-${getServiceTone(status)}`}>
      {formatServiceStatus(status)}
    </span>
  );
}

export function TaskStatePill({ status }: { status: ControlTaskStatus }) {
  return <span className={`task-state task-state-${status}`}>{taskStatusLabel[status]}</span>;
}

export function RuntimeApplyPipeline({
  apply,
  compact = false
}: {
  apply: RuntimeApplyView;
  compact?: boolean;
}) {
  return (
    <div
      className={`runtime-pipeline${compact ? " compact" : ""}`}
      aria-label="Runtime apply stages"
    >
      {apply.phases.map((phase) => (
        <span
          className={`runtime-stage runtime-stage-${phase.status}`}
          key={phase.stage}
          title={`${runtimeApplyStageLabel[phase.stage]} ${phase.status}`}
        >
          {compact
            ? runtimeApplyStageLabel[phase.stage].slice(0, 1)
            : runtimeApplyStageLabel[phase.stage]}
        </span>
      ))}
    </div>
  );
}

function RuntimeApplySummary({ apply }: { apply: RuntimeApplyView }) {
  return (
    <div className="runtime-apply-summary">
      <div className="runtime-apply-head">
        <div>
          <span>Runtime apply</span>
          <strong>{runtimeApplyStageLabel[apply.currentStage]}</strong>
        </div>
        <ServiceStatusPill status={apply.serviceStatus} />
      </div>
      <RuntimeApplyPipeline apply={apply} />
      <div className="runtime-apply-meta">
        <RuntimeMeta label="Config" value={apply.configPath} />
        <RuntimeMeta
          label="Rollback"
          value={apply.rollbackAvailable ? "Available" : "Unavailable"}
        />
        <RuntimeFailureStage stage={apply.failureStage} />
      </div>
    </div>
  );
}

function RuntimeMeta({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span>{label}</span>
      <strong title={value}>{value}</strong>
    </div>
  );
}

function RuntimeFailureStage({ stage }: { stage?: RuntimeApplyStage }) {
  if (!stage) {
    return null;
  }

  return (
    <span className="failure-stage">
      Failed at {runtimeApplyStageLabel[stage]}
    </span>
  );
}

function ControlDatum({
  label,
  value,
  children
}: {
  label: string;
  value?: string;
  children?: ReactNode;
}) {
  return (
    <div>
      <span>{label}</span>
      {children ?? <strong>{value}</strong>}
    </div>
  );
}

function CapabilityList({
  capabilities,
  compact = false
}: {
  capabilities: string[];
  compact?: boolean;
}) {
  const visibleCapabilities = compact ? capabilities.slice(0, 2) : capabilities;
  const overflowCount = capabilities.length - visibleCapabilities.length;

  return (
    <div className={`capability-list${compact ? " compact" : ""}`}>
      {visibleCapabilities.length > 0 ? (
        <>
          {visibleCapabilities.map((capability) => (
            <span className="capability-chip" key={capability}>
              {capability}
            </span>
          ))}
          {overflowCount > 0 ? (
            <span className="capability-chip">+{overflowCount}</span>
          ) : null}
        </>
      ) : (
        <span className="capability-chip muted">Not reported</span>
      )}
    </div>
  );
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
  const width = Math.min(Math.max(value, 0), 100);

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
        <span style={{ width: `${width}%` }} />
      </div>
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

function getServiceTone(status: string): "ok" | "warning" | "danger" | "muted" | "info" {
  const normalized = status.toLowerCase();

  if (["running", "active", "healthy", "ok"].some((term) => normalized.includes(term))) {
    return "ok";
  }
  if (["reload", "starting", "pending", "maintenance"].some((term) => normalized.includes(term))) {
    return "warning";
  }
  if (["degraded", "failed", "error", "stopped", "offline"].some((term) => normalized.includes(term))) {
    return "danger";
  }
  if (["unknown", "unreported"].some((term) => normalized.includes(term))) {
    return "muted";
  }

  return "info";
}

import {
  getAgentHostTuning,
  getAgentRuntimeApply,
  getAgentTaskState,
  getAuthState,
  getLastHeartbeat,
  getRegistrationState,
  getRuntimeCapabilities,
  getRuntimeLabel,
  hostTuneStageLabel,
  runtimeApplyStageLabel,
  type ControlBadgeState,
  type HostTuningView,
  type RuntimeApplyView
} from "../controlFields";
import type {
  Agent,
  AgentStatus,
  ControlTaskStatus,
  HostTuneStage,
  RuntimeApplyStage
} from "../data";
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
          const hostTuning = getAgentHostTuning(agent);

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
                <span>{formatServiceMode(runtimeApply.serviceMode)}</span>
                <span>{formatRuntimeManaged(runtimeApply.runtimeManaged)}</span>
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

              <RuntimeServiceSummary apply={runtimeApply} />
              <HostTuningSummary tuning={hostTuning} />

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
              <th>Control stages</th>
              <th>Host tuning</th>
              <th>Unit / config</th>
              <th>Service actions</th>
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
              const hostTuning = getAgentHostTuning(agent);

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
                      <span>{formatServiceMode(runtimeApply.serviceMode)}</span>
                      <span>{formatRuntimeManaged(runtimeApply.runtimeManaged)}</span>
                      <ServiceStatusPill status={runtimeApply.serviceStatus} />
                    </div>
                  </td>
                  <td>
                    <RuntimeApplyPipeline apply={runtimeApply} compact />
                    <RuntimeFailureStage stage={runtimeApply.failureStage} />
                  </td>
                  <td>
                    <HostTuningTableCell tuning={hostTuning} />
                  </td>
                  <td>
                    <div className="config-cell">
                      <span title={runtimeApply.unitPath}>{runtimeApply.unitPath}</span>
                      <span title={runtimeApply.configDir}>{runtimeApply.configDir}</span>
                      <span title={runtimeApply.configPath}>{runtimeApply.configPath}</span>
                      <strong>
                        {runtimeApply.rollbackAvailable
                          ? "Rollback available"
                          : "Rollback unavailable"}
                      </strong>
                    </div>
                  </td>
                  <td>
                    <RuntimeControlSignals apply={runtimeApply} compact />
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
      aria-label="Runtime service control stages"
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

function HostTuningSummary({ tuning }: { tuning: HostTuningView }) {
  return (
    <div className="host-tuning-summary">
      <div className="host-tuning-head">
        <div>
          <span>Host tuning / Network optimization</span>
          <strong>{hostTuneStageLabel[tuning.currentStage]}</strong>
        </div>
        <TaskStatePill status={tuning.status} />
      </div>
      <HostTunePipeline tuning={tuning} />
      <div className="host-tuning-meta">
        <RuntimeMeta label="BBR" value={`${tuning.bbrStatus} / ${tuning.bbrVersion}`} />
        <RuntimeMeta label="Sysctl profile" value={tuning.sysctlProfile} />
        <RuntimeMeta label="Kernel" value={tuning.kernelVersion} />
        <RuntimeMeta
          label="Congestion"
          value={`${tuning.currentCongestionControl} -> ${tuning.targetCongestionControl}`}
        />
        <RuntimeMeta label="Reboot" value={tuning.rebootRequired ? "Required" : "Not required"} />
        <RuntimeMeta label="One-click task" value={`${tuning.taskId} / ${tuning.eta}`} />
        <HostTuneFailureStage stage={tuning.failureStage} />
      </div>
    </div>
  );
}

function HostTuningTableCell({ tuning }: { tuning: HostTuningView }) {
  return (
    <div className="host-tuning-cell">
      <div className="host-tuning-cell-head">
        <TaskStatePill status={tuning.status} />
        <span>{tuning.rebootRequired ? "Reboot required" : "No reboot"}</span>
      </div>
      <HostTunePipeline tuning={tuning} compact />
      <span>{tuning.bbrStatus}</span>
      <span>{tuning.sysctlProfile}</span>
      <span>
        {tuning.taskId} / {tuning.eta}
      </span>
      <strong title={tuning.kernelVersion}>{tuning.kernelVersion}</strong>
      <span>
        {tuning.currentCongestionControl} -&gt; {tuning.targetCongestionControl}
      </span>
      <HostTuneFailureStage stage={tuning.failureStage} />
    </div>
  );
}

function HostTunePipeline({
  tuning,
  compact = false
}: {
  tuning: HostTuningView;
  compact?: boolean;
}) {
  return (
    <div
      className={`host-tune-pipeline${compact ? " compact" : ""}`}
      aria-label="Host tuning stages"
    >
      {tuning.phases.map((phase) => (
        <span
          className={`runtime-stage runtime-stage-${phase.status}`}
          key={phase.stage}
          title={`${hostTuneStageLabel[phase.stage]} ${phase.status}`}
        >
          {compact ? hostTuneStageLabel[phase.stage].slice(0, 1) : hostTuneStageLabel[phase.stage]}
        </span>
      ))}
    </div>
  );
}

function RuntimeServiceSummary({ apply }: { apply: RuntimeApplyView }) {
  return (
    <div className="runtime-apply-summary">
      <div className="runtime-apply-head">
        <div>
          <span>Service control</span>
          <strong>{runtimeApplyStageLabel[apply.currentStage]}</strong>
        </div>
        <ServiceStatusPill status={apply.serviceStatus} />
      </div>
      <RuntimeApplyPipeline apply={apply} />
      <div className="runtime-apply-meta">
        <RuntimeMeta label="Mode" value={formatServiceMode(apply.serviceMode)} />
        <RuntimeMeta label="Managed" value={formatRuntimeManaged(apply.runtimeManaged)} />
        <RuntimeMeta label="Unit" value={apply.unitPath} />
        <RuntimeMeta label="Config dir" value={apply.configDir} />
        <RuntimeMeta label="Config" value={apply.configPath} />
        <RuntimeMeta
          label="Rollback"
          value={apply.rollbackAvailable ? "Available" : "Unavailable"}
        />
        <RuntimeFailureStage stage={apply.failureStage} />
      </div>
      <RuntimeControlSignals apply={apply} />
    </div>
  );
}

function RuntimeControlSignals({
  apply,
  compact = false
}: {
  apply: RuntimeApplyView;
  compact?: boolean;
}) {
  return (
    <div className={`runtime-control-signals${compact ? " compact" : ""}`}>
      <RuntimeSignal label="Reload" status={apply.reloadStatus} detail={apply.reloadInfo} />
      <RuntimeSignal label="Restart" status={apply.restartStatus} detail={apply.restartInfo} />
      <RuntimeSignal label="Health" status={apply.healthStatus} detail={apply.healthInfo} />
    </div>
  );
}

function RuntimeSignal({
  label,
  status,
  detail
}: {
  label: string;
  status: string;
  detail: string;
}) {
  return (
    <div>
      <span>{label}</span>
      <strong title={detail}>{formatServiceStatus(status)}</strong>
      <small title={detail}>{detail}</small>
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

function HostTuneFailureStage({ stage }: { stage?: HostTuneStage }) {
  if (!stage) {
    return null;
  }

  return (
    <span className="failure-stage">
      Failed at {hostTuneStageLabel[stage]}
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

function formatServiceMode(mode: string): string {
  return `${formatServiceStatus(mode)} mode`;
}

function formatRuntimeManaged(managed: boolean): string {
  return managed ? "OU-UI managed" : "External service";
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

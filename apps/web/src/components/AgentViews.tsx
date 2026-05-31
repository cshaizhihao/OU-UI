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
import { useLocale } from "./ConsolePrimitives";

const statusLabel: Record<AgentStatus, string> = {
  online: "在线",
  degraded: "降级",
  offline: "离线"
};

const englishStatusLabel: Record<AgentStatus, string> = {
  online: "Online",
  degraded: "Degraded",
  offline: "Offline"
};

const taskStatusLabel: Record<ControlTaskStatus, string> = {
  pending: "等待中",
  running: "执行中",
  success: "成功",
  failed: "失败"
};

const englishTaskStatusLabel: Record<ControlTaskStatus, string> = {
  pending: "Pending",
  running: "Running",
  success: "Success",
  failed: "Failed"
};

type AgentViewsProps = {
  agents: Agent[];
};

export function ProbeAgentCards({ agents }: AgentViewsProps) {
  const language = useLocale();
  const isZh = language === "zh-CN";
  return (
    <section className="panel probe-panel" id="agent-probes">
      <div className="section-heading compact">
        <div>
          <p className="eyebrow">{isZh ? "Agent 探针" : "Agent probes"}</p>
          <h2>{isZh ? "主机状态" : "Host status"}</h2>
        </div>
        <button className="ghost-button" type="button">{isZh ? "探针同步" : "Sync probes"}</button>
      </div>
      <div className="probe-grid">
        {agents.map((agent) => {
          const runtime = getRuntimeLabel(agent.runtime);
          const runtimeApply = getAgentRuntimeApply(agent);
          const task = getAgentTaskState(agent);
          const trafficPercent = quotaPercent(agent.usedTrafficGb, agent.quotaTrafficGb);
          const totalMbps = Number(agent.uplinkMbps || 0) + Number(agent.downlinkMbps || 0);
          const diskPercent = agent.diskPercent ?? quotaPercent(agent.diskUsedGb ?? 0, agent.diskTotalGb ?? 0);
          const latency = formatLatency(agent, language);
          const loss = formatLoss(agent, language);

          return (
            <article className={`probe-card probe-${agent.status}`} key={agent.id}>
              <div className="probe-head">
                <div className="probe-title">
                  <span className="probe-flag">{agent.region?.slice(0, 1) || "OU"}</span>
                  <div>
                    <strong>{agent.name}</strong>
                    <span>{agent.region || agent.ip || agent.id}</span>
                  </div>
                </div>
                <div className="probe-badges">
                  <span>{runtime}</span>
                  <span>IPv4</span>
                  <i aria-label={isZh ? statusLabel[agent.status] : englishStatusLabel[agent.status]} />
                </div>
              </div>

              <div className="probe-resource-grid">
                <ProbeMeter label="CPU" value={agent.cpu} detail={formatCpuDetail(agent, language)} />
                <ProbeMeter label={isZh ? "内存" : "Memory"} value={agent.memory} detail={`${agent.memory.toFixed(2)}%`} tone="violet" />
                <ProbeMeter
                  label={isZh ? "磁盘" : "Disk"}
                  value={diskPercent}
                  detail={formatDiskDetail(agent, language)}
                  tone="cyan"
                />
                <ProbeMeter
                  label={isZh ? "月度" : "Monthly"}
                  value={trafficPercent}
                  detail={`${agent.usedTrafficGb}G / ${agent.quotaTrafficGb || "∞"}G`}
                  tone="blue"
                />
              </div>

              <div className="probe-traffic">
                <ProbeTraffic label={isZh ? "下载" : "Down"} value={agent.downlinkMbps} unit="Mbps" direction="down" />
                <ProbeTraffic label={isZh ? "上传" : "Up"} value={agent.uplinkMbps} unit="Mbps" direction="up" />
              </div>

              <div className="probe-mini-row">
                <span>{isZh ? "累计" : "Total"} {agent.usedTrafficGb} GB</span>
                <span>{isZh ? "总速率" : "Total rate"} {totalMbps.toFixed(2)} Mbps</span>
              </div>

              <div className="probe-quality">
                <ProbeQuality label={isZh ? "延迟" : "Latency"} value={latency.value} danger={latency.danger} />
                <ProbeQuality label={isZh ? "丢包率" : "Loss"} value={loss.value} danger={loss.danger} />
              </div>

              <div className="probe-stripes" aria-hidden="true">
                {Array.from({ length: 28 }).map((_, index) => (
                  <span className={stripeClass(agent.status, index)} key={index} />
                ))}
              </div>

              <div className="probe-foot">
                <span>{isZh ? "到期" : "Expires"} {formatExpiry(agent.expiresInDays, language)}</span>
                <span>{isZh ? "在线" : "Online"} {formatUptime(agent, language)}</span>
                <span>{isZh ? "任务" : "Task"} {isZh ? taskStatusLabel[task.status] : englishTaskStatusLabel[task.status]}</span>
                <span>{isZh ? "服务" : "Service"} {formatServiceStatus(runtimeApply.serviceStatus)}</span>
              </div>
            </article>
          );
        })}
        {agents.length === 0 ? <p className="empty-state">{isZh ? "暂无 Agent 数据" : "No Agent data"}</p> : null}
      </div>
    </section>
  );
}

export function AgentCards({ agents }: AgentViewsProps) {
  return (
    <section className="panel" id="agents">
      <div className="section-heading">
        <div>
          <p className="eyebrow">Agent 监控</p>
          <h2>代理节点监控</h2>
        </div>
        <button className="ghost-button">同步探测</button>
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
                <span>队列 {agent.queue}</span>
              </div>

              <div className="control-summary" aria-label="Agent control chain">
                <ControlDatum label="注册">
                  <ControlPill state={registration} />
                </ControlDatum>
                <ControlDatum label="认证">
                  <ControlPill state={auth} />
                </ControlDatum>
                <ControlDatum label="最后心跳" value={heartbeat} />
              </div>

              <RuntimeServiceSummary apply={runtimeApply} />
              <HostTuningSummary tuning={hostTuning} />

              <div className="agent-task-summary">
                <div>
                  <span>任务</span>
                  <TaskStatePill status={task.status} />
                </div>
                <div>
                  <span>重试</span>
                  <strong>{task.retryCount}</strong>
                </div>
                {task.failureReason ? (
                  <p className="failure-reason">失败原因：{task.failureReason}</p>
                ) : null}
              </div>

              <CapabilityList capabilities={capabilities} />

              <div className="agent-bars">
                <MeterLine label="CPU" value={agent.cpu} suffix="%" />
                <MeterLine label="Memory" value={agent.memory} suffix="%" />
              </div>

              <div className="agent-metrics">
                <Metric label="上传" value={`${agent.uplinkMbps} Mbps`} />
                <Metric label="下载" value={`${agent.downlinkMbps} Mbps`} />
                <Metric
                  label="流量"
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

function ProbeMeter({
  detail,
  label,
  tone = "green",
  value
}: {
  detail: string;
  label: string;
  tone?: "blue" | "cyan" | "green" | "violet";
  value: number;
}) {
  const width = Math.min(Math.max(Number(value) || 0, 0), 100);
  return (
    <div className="probe-meter">
      <div>
        <span>{label}</span>
        <strong>{detail}</strong>
      </div>
      <div className={`probe-track probe-track-${tone}`}>
        <span style={{ width: `${width}%` }} />
      </div>
    </div>
  );
}

function ProbeTraffic({
  direction,
  label,
  unit,
  value
}: {
  direction: "down" | "up";
  label: string;
  unit: string;
  value: number;
}) {
  return (
    <div className={`probe-flow probe-flow-${direction}`}>
      <span>{direction === "down" ? "↓" : "↑"} {label}</span>
      <strong>
        {Number(value || 0).toFixed(2)} <small>{unit}</small>
      </strong>
    </div>
  );
}

function ProbeQuality({ danger = false, label, value }: { danger?: boolean; label: string; value: string }) {
  return (
    <div>
      <span>{label}</span>
      <strong className={danger ? "danger" : ""}>{value}</strong>
    </div>
  );
}

function quotaPercent(used: number, quota: number): number {
  if (!Number.isFinite(quota) || quota <= 0) {
    return Math.min(Math.max(used > 0 ? 12 : 0, 0), 100);
  }
  return Math.min(Math.max((used / quota) * 100, 0), 100);
}

function formatCpuDetail(agent: Agent, language: "zh-CN" | "en"): string {
  const percent = `${agent.cpu.toFixed(2)}%`;
  if (!agent.cpuCores) {
    return percent;
  }
  return language === "zh-CN" ? `${percent} / ${agent.cpuCores}核` : `${percent} / ${agent.cpuCores} cores`;
}

function formatDiskDetail(agent: Agent, language: "zh-CN" | "en"): string {
  if (!agent.diskTotalGb || agent.diskTotalGb <= 0) {
    return language === "zh-CN" ? "未上报" : "Not reported";
  }
  const used = agent.diskUsedGb ?? 0;
  return `${used}G / ${agent.diskTotalGb}G`;
}

function formatLatency(agent: Agent, language: "zh-CN" | "en"): { value: string; danger: boolean } {
  if (typeof agent.latencyMs === "number" && Number.isFinite(agent.latencyMs)) {
    return { value: `${Math.round(agent.latencyMs)} ms`, danger: agent.latencyMs >= 180 };
  }
  if (agent.status === "offline") {
    return { value: language === "zh-CN" ? "不可达" : "Unreachable", danger: true };
  }
  return { value: "-- ms", danger: false };
}

function formatLoss(agent: Agent, language: "zh-CN" | "en"): { value: string; danger: boolean } {
  if (typeof agent.lossPercent === "number" && Number.isFinite(agent.lossPercent)) {
    return { value: `${agent.lossPercent.toFixed(1)}%`, danger: agent.lossPercent >= 2 };
  }
  if (agent.status === "offline") {
    return { value: "100%", danger: true };
  }
  return { value: language === "zh-CN" ? "未上报" : "N/A", danger: false };
}

function formatExpiry(value: number | null | undefined, language: "zh-CN" | "en"): string {
  if (value === null) {
    return language === "zh-CN" ? "永久" : "Permanent";
  }
  if (typeof value === "number" && Number.isFinite(value)) {
    return language === "zh-CN" ? `${value}天` : `${value}d`;
  }
  return language === "zh-CN" ? "永久" : "Permanent";
}

function formatUptime(agent: Agent, language: "zh-CN" | "en"): string {
  if (agent.uptimeSeconds && agent.uptimeSeconds > 0) {
    const days = Math.floor(agent.uptimeSeconds / 86400);
    if (days > 0) {
      return language === "zh-CN" ? `${days}天` : `${days}d`;
    }
    const hours = Math.floor(agent.uptimeSeconds / 3600);
    if (hours > 0) {
      return language === "zh-CN" ? `${hours}小时` : `${hours}h`;
    }
  }
  return formatProbeAge(agent.lastHeartbeatAt || agent.updatedAt, language);
}

function stripeClass(status: AgentStatus, index: number): string {
  if (status === "offline") {
    return index % 6 === 0 ? "danger" : "muted";
  }
  if (status === "degraded") {
    return index % 8 === 0 ? "warn" : "";
  }
  return "";
}

function formatProbeAge(value: string, language: "zh-CN" | "en"): string {
  const relative = parseRelativeAge(value);
  if (relative !== undefined) {
    return relativeAgeLabel(relative, language);
  }
  const time = new Date(value).getTime();
  if (!Number.isFinite(time)) {
    return value && value !== "not seen" ? value : "--";
  }
  const minutes = Math.max(0, Math.round((Date.now() - time) / 60000));
  return relativeAgeLabel(minutes, language);
}

function parseRelativeAge(value: string): number | undefined {
  const match = value?.trim().match(/^(\d+)\s*([smhd])\s*ago$/i);
  if (!match) {
    return undefined;
  }
  const amount = Number(match[1]);
  if (!Number.isFinite(amount)) {
    return undefined;
  }
  switch (match[2].toLowerCase()) {
    case "s":
      return Math.max(0, Math.round(amount / 60));
    case "m":
      return amount;
    case "h":
      return amount * 60;
    case "d":
      return amount * 1440;
    default:
      return undefined;
  }
}

function relativeAgeLabel(minutes: number, language: "zh-CN" | "en"): string {
  if (minutes < 60) {
    return language === "zh-CN" ? `${minutes} 分钟` : `${minutes} min`;
  }
  const hours = Math.round(minutes / 60);
  if (hours < 48) {
    return language === "zh-CN" ? `${hours} 小时` : `${hours} h`;
  }
  const days = Math.round(hours / 24);
  return language === "zh-CN" ? `${days} 天` : `${days} d`;
}

export function AgentTable({ agents }: AgentViewsProps) {
  return (
    <section className="panel">
      <div className="section-heading compact">
        <h2>Agent 明细</h2>
        <div className="segmented" aria-label="Agent status filter">
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
              <th>控制</th>
              <th>运行时</th>
              <th>部署阶段</th>
              <th>主机调优</th>
              <th>Unit / 配置</th>
              <th>服务动作</th>
              <th>能力</th>
              <th>CPU</th>
              <th>内存</th>
              <th>上传 / 下载</th>
              <th>流量配额</th>
              <th>心跳</th>
              <th>任务</th>
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
                        {runtimeApply.rollbackAvailable ? "可回滚" : "不可回滚"}
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
                      <span>重试 {task.retryCount}</span>
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
          <span>主机调优 / 网络优化</span>
          <strong>{hostTuneStageLabel[tuning.currentStage]}</strong>
        </div>
        <TaskStatePill status={tuning.status} />
      </div>
      <HostTunePipeline tuning={tuning} />
      <div className="host-tuning-meta">
        <RuntimeMeta label="BBR" value={`${tuning.bbrStatus} / ${tuning.bbrVersion}`} />
        <RuntimeMeta label="Sysctl 配置" value={tuning.sysctlProfile} />
        <RuntimeMeta label="内核" value={tuning.kernelVersion} />
        <RuntimeMeta
          label="拥塞控制"
          value={`${tuning.currentCongestionControl} -> ${tuning.targetCongestionControl}`}
        />
        <RuntimeMeta label="重启" value={tuning.rebootRequired ? "需要" : "不需要"} />
        <RuntimeMeta label="一键任务" value={`${tuning.taskId} / ${tuning.eta}`} />
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
        <span>{tuning.rebootRequired ? "需要重启" : "无需重启"}</span>
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
          <span>服务控制</span>
          <strong>{runtimeApplyStageLabel[apply.currentStage]}</strong>
        </div>
        <ServiceStatusPill status={apply.serviceStatus} />
      </div>
      <RuntimeApplyPipeline apply={apply} />
      <div className="runtime-apply-meta">
        <RuntimeMeta label="模式" value={formatServiceMode(apply.serviceMode)} />
        <RuntimeMeta label="托管" value={formatRuntimeManaged(apply.runtimeManaged)} />
        <RuntimeMeta label="Unit" value={apply.unitPath} />
        <RuntimeMeta label="配置目录" value={apply.configDir} />
        <RuntimeMeta label="配置文件" value={apply.configPath} />
        <RuntimeMeta
          label="回滚"
          value={apply.rollbackAvailable ? "可用" : "不可用"}
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
      <RuntimeSignal label="健康" status={apply.healthStatus} detail={apply.healthInfo} />
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
      失败阶段：{runtimeApplyStageLabel[stage]}
    </span>
  );
}

function HostTuneFailureStage({ stage }: { stage?: HostTuneStage }) {
  if (!stage) {
    return null;
  }

  return (
    <span className="failure-stage">
      失败阶段：{hostTuneStageLabel[stage]}
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
        <span className="capability-chip muted">未上报</span>
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
  return `${formatServiceStatus(mode)} 模式`;
}

function formatRuntimeManaged(managed: boolean): string {
  return managed ? "OU-UI 托管" : "外部服务";
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

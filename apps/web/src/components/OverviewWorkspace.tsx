import type { DashboardDTO } from "../api";
import type { Agent } from "../data";
import { ProbeAgentCards } from "./AgentViews";
import {
  formatBytes,
  formatServiceStatus,
  KpiGrid,
  MiniTable,
  SectionHeader,
  taskTone,
  useFormatTime,
  ViewHeading
} from "./ConsolePrimitives";

type OverviewWorkspaceProps = {
  agents: Agent[];
  data: DashboardDTO | null;
  onRefresh?: () => void;
};

export function OverviewWorkspace({ agents, data, onRefresh }: OverviewWorkspaceProps) {
  const formatTime = useFormatTime();
  const control = data?.control;
  const trafficBytes =
    control?.traffic.reduce((sum, item) => sum + item.rxBytes + item.txBytes, 0) ?? 0;
  const connections = control?.traffic.reduce((sum, item) => sum + item.connections, 0) ?? 0;
  const onlineAgents = data?.overview.agentsOnline ?? agents.filter((agent) => agent.status === "online").length;
  const nodeTotal = data?.overview.nodesTotal ?? control?.nodes.length ?? 0;
  const openAlerts = control?.alerts.filter((alert) => !alert.delivered).length ?? 0;

  return (
    <div className="workspace-view">
      <ViewHeading
        actions={
          <button className="ghost-button" onClick={onRefresh} type="button">
            刷新快照
          </button>
        }
        description="这里保留控制面全局态势，不再混放所有配置表单。"
        eyebrow="总览"
        title="运行态势"
      />

      <KpiGrid
        items={[
          {
            label: "在线 Agent",
            value: `${onlineAgents} / ${data?.overview.agentsTotal ?? agents.length}`,
            delta: data?.overview.version ?? "等待实时快照"
          },
          { label: "托管节点", value: String(nodeTotal), delta: "自建与外部节点统一管理" },
          { label: "单节点流量", value: formatBytes(trafficBytes), delta: `${connections} 活跃连接` },
          { label: "开放告警", value: String(openAlerts), delta: "Webhook 自动投递" }
        ]}
      />

      <div className="overview-command-grid">
        <ProbeAgentCards agents={agents.slice(0, 6)} />
        <div className="overview-side-rail">
          <section className="panel">
            <SectionHeader eyebrow="任务" title="最近执行队列" />
            <div className="task-list">
              {(control?.tasks ?? []).slice(0, 5).map((task) => {
                const tone = taskTone(task.status);
                return (
                  <article className="task-item" key={task.id}>
                    <div className="task-item-head">
                      <div>
                        <strong>{task.type}</strong>
                        <span>{task.agentId || "未绑定 Agent"}</span>
                      </div>
                      <small>{formatTime(task.updatedAt ?? task.createdAt)}</small>
                    </div>
                    <div className="progress">
                      <span className={`progress-${tone}`} style={{ width: tone === "success" || tone === "failed" ? "100%" : "48%" }} />
                    </div>
                    <div className="task-meta">
                      <span>{task.id}</span>
                      <span className={`task-state task-state-${tone}`}>{formatServiceStatus(task.status)}</span>
                    </div>
                  </article>
                );
              })}
              {(control?.tasks.length ?? 0) === 0 ? <p className="empty-state">暂无任务</p> : null}
            </div>
          </section>

          <section className="panel">
            <SectionHeader eyebrow="告警" title="最新事件" />
            <MiniTable
              columns={["事件", "来源", "状态", "时间"]}
              rows={(control?.alerts ?? []).slice(0, 6).map((alert) => [
                alert.eventType,
                `${alert.sourceType}:${alert.sourceId}`,
                alert.delivered ? "已投递" : alert.lastError || "待处理",
                formatTime(alert.createdAt)
              ])}
            />
          </section>
        </div>
      </div>
    </div>
  );
}

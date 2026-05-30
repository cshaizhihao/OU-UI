import { useEffect, useMemo, useState } from "react";
import type { DashboardDTO, ManagedNode, NodeTraffic } from "../api";
import type { Agent } from "../data";
import { protocolOptions, runtimeOptions } from "../data";
import { AgentTable, ProbeAgentCards } from "./AgentViews";
import {
  formatBytes,
  formatServiceStatus,
  KpiGrid,
  MiniTable,
  SectionHeader,
  StatusTag,
  useFormatTime,
  ViewHeading
} from "./ConsolePrimitives";

type NodeManagementWorkspaceProps = {
  agents: Agent[];
  data: DashboardDTO | null;
};

export function NodeManagementWorkspace({ agents, data }: NodeManagementWorkspaceProps) {
  const formatTime = useFormatTime();
  const nodes = data?.control.nodes ?? [];
  const traffic = data?.control.traffic ?? [];
  const onlineAgents = agents.filter((agent) => agent.status === "online").length;
  const activeNodes = nodes.filter((node) => !["failed", "offline", "disabled"].includes(node.status)).length;
  const queuedTasks = data?.control.tasks.filter((task) => ["pending", "queued", "running"].includes(task.status)).length ?? 0;
  const nodeInsights = useMemo(() => buildNodeInsights(nodes, traffic, agents), [nodes, traffic, agents]);
  const [selectedNodeId, setSelectedNodeId] = useState("");
  const selectedInsight = nodeInsights.find((item) => item.node.id === selectedNodeId) ?? nodeInsights[0];

  useEffect(() => {
    if (!selectedNodeId && nodeInsights[0]) {
      setSelectedNodeId(nodeInsights[0].node.id);
      return;
    }
    if (selectedNodeId && !nodeInsights.some((item) => item.node.id === selectedNodeId)) {
      setSelectedNodeId(nodeInsights[0]?.node.id ?? "");
    }
  }, [nodeInsights, selectedNodeId]);

  return (
    <div className="workspace-view">
      <ViewHeading
        description="这里集中处理 Agent、生成节点、运行时部署和服务状态，不再和其他业务上下堆叠。"
        eyebrow="节点管理"
        title="Agent 与托管节点"
      />

      <KpiGrid
        items={[
          { label: "在线 Agent", value: `${onlineAgents} / ${agents.length}`, delta: "心跳在线" },
          { label: "健康节点", value: `${activeNodes} / ${nodes.length}`, delta: "运行时状态" },
          { label: "任务队列", value: String(queuedTasks), delta: "部署与调优任务" },
          { label: "协议能力", value: String(protocolOptions.length), delta: runtimeOptions.join(" / ") }
        ]}
      />

      <section className="panel">
        <SectionHeader eyebrow="节点下发" title="运行时服务控制" />
        <form className="dispatch-form">
          <label>
            Agent
            <select defaultValue={agents[0]?.id ?? ""} disabled={!agents.length}>
              {agents.length ? (
                agents.map((agent) => (
                  <option value={agent.id} key={agent.id}>
                    {agent.name} - {agent.region}
                  </option>
                ))
              ) : (
                <option value="">暂无可用 Agent</option>
              )}
            </select>
          </label>
          <label>
            运行时
            <select defaultValue="Xray">
              {runtimeOptions.map((runtime) => (
                <option value={runtime} key={runtime}>
                  {runtime}
                </option>
              ))}
            </select>
          </label>
          <label>
            协议
            <select defaultValue="VLESS Reality">
              {protocolOptions.map((protocol) => (
                <option value={protocol} key={protocol}>
                  {protocol}
                </option>
              ))}
            </select>
          </label>
          <label>
            队列策略
            <select defaultValue="rolling">
              <option value="rolling">托管 reload，保持活动连接</option>
              <option value="immediate">托管 restart，维护窗口执行</option>
              <option value="staged">外部服务，等待人工确认</option>
            </select>
          </label>
        </form>
      </section>

      <section className="panel node-inspector-panel">
        <SectionHeader eyebrow="Node Desk" title="节点详情与订阅可用性" />
        <div className="node-inspector-layout">
          <div className="node-inspector-list" role="listbox" aria-label="托管节点选择">
            {nodeInsights.map((item) => (
              <button
                aria-selected={selectedInsight?.node.id === item.node.id}
                className={selectedInsight?.node.id === item.node.id ? "selected" : ""}
                key={item.node.id}
                onClick={() => setSelectedNodeId(item.node.id)}
                type="button"
              >
                <span className={`node-health-dot node-health-${item.tone}`} aria-hidden="true" />
                <div>
                  <strong>{item.node.name}</strong>
                  <span>{item.node.protocol} / {item.agent?.name ?? item.node.agentId}</span>
                </div>
                <em>{item.rateMbps} Mbps</em>
              </button>
            ))}
            {nodeInsights.length === 0 ? <p className="empty-state">暂无托管节点</p> : null}
          </div>
          {selectedInsight ? (
            <NodeInspector insight={selectedInsight} formatTime={formatTime} />
          ) : (
            <div className="node-inspector-empty">
              <strong>等待节点接入</strong>
              <span>Agent 上报托管节点后，这里会显示运行时、流量、订阅和服务状态。</span>
            </div>
          )}
        </div>
      </section>

      <ProbeAgentCards agents={agents} />

      <section className="panel">
        <SectionHeader eyebrow="托管节点" title="节点健康" />
        <MiniTable
          columns={["节点", "Agent", "运行时", "协议", "状态"]}
          rows={nodes.slice(0, 10).map((node) => [
            node.name,
            node.agentId,
            node.runtime,
            node.protocol,
            formatServiceStatus(node.serviceStatus || node.status)
          ])}
        />
      </section>

      <AgentTable agents={agents} />
    </div>
  );
}

type NodeInsight = {
  node: ManagedNode;
  agent?: Agent;
  samples: NodeTraffic[];
  latest?: NodeTraffic;
  rxBytes: number;
  txBytes: number;
  rateMbps: string;
  connections: number;
  tone: "ok" | "warning" | "danger" | "muted";
};

function NodeInspector({
  formatTime,
  insight
}: {
  formatTime: (value?: string) => string;
  insight: NodeInsight;
}) {
  const serviceStatus = insight.node.serviceStatus || insight.node.status;
  const subscriptionReady = isSubscriptionReady(insight);
  return (
    <div className="node-inspector-detail">
      <div className="node-inspector-head">
        <div>
          <span>{insight.node.id}</span>
          <h3>{insight.node.name}</h3>
        </div>
        <StatusTag tone={insight.tone}>{formatServiceStatus(serviceStatus)}</StatusTag>
      </div>

      <div className="node-inspector-metrics">
        <MetricTile label="实时速率" value={`${insight.rateMbps} Mbps`} detail={`${insight.connections} 连接`} />
        <MetricTile label="累计上传" value={formatBytes(insight.rxBytes)} detail={insight.samples.length ? `${insight.samples.length} 个样本` : "暂无样本"} />
        <MetricTile label="累计下载" value={formatBytes(insight.txBytes)} detail={formatTime(insight.latest?.collectedAt)} />
      </div>

      <div className="node-inspector-sections">
        <div>
          <span>运行时</span>
          <strong>{insight.node.runtime} / {insight.node.protocol}</strong>
          <small>{insight.node.configPath || "配置路径未上报"}</small>
        </div>
        <div>
          <span>Agent</span>
          <strong>{insight.agent?.name ?? insight.node.agentId}</strong>
          <small>{insight.agent?.region || insight.agent?.ip || "未绑定实时 Agent"}</small>
        </div>
        <div>
          <span>订阅状态</span>
          <strong>{subscriptionReady ? "可纳入聚合订阅" : "等待健康检查"}</strong>
          <small>{subscriptionReady ? "Clash / V2Ray / Sing-box 可复用该节点" : "需确认服务可用、节点在线或有采样"}</small>
        </div>
        <div>
          <span>最近变更</span>
          <strong>{formatTime(insight.node.updatedAt)}</strong>
          <small>{insight.node.lastError || "无最近错误"}</small>
        </div>
      </div>

      <div className="node-action-row" aria-label="节点快捷操作">
        <button className="ghost-button" type="button">复制订阅标识</button>
        <button className="ghost-button" type="button">查看流量样本</button>
        <button className="ghost-button" type="button">生成 Clash 片段</button>
      </div>
    </div>
  );
}

function MetricTile({ detail, label, value }: { detail: string; label: string; value: string }) {
  return (
    <div>
      <span>{label}</span>
      <strong>{value}</strong>
      <small>{detail}</small>
    </div>
  );
}

function buildNodeInsights(nodes: ManagedNode[], traffic: NodeTraffic[], agents: Agent[]): NodeInsight[] {
  return nodes.map((node) => {
    const samples = traffic.filter((sample) => sample.nodeId === node.id);
    const latest = [...samples].sort((a, b) => new Date(b.collectedAt).getTime() - new Date(a.collectedAt).getTime())[0];
    const rxBytes = samples.reduce((sum, sample) => sum + sample.rxBytes, 0);
    const txBytes = samples.reduce((sum, sample) => sum + sample.txBytes, 0);
    const rateBps = (latest?.rxRateBps ?? 0) + (latest?.txRateBps ?? 0);
    const serviceStatus = (node.serviceStatus || node.status || "").toLowerCase();
    const agent = agents.find((item) => item.id === node.agentId);
    return {
      node,
      agent,
      samples,
      latest,
      rxBytes,
      txBytes,
      rateMbps: toMbps(rateBps),
      connections: latest?.connections ?? 0,
      tone: nodeTone(serviceStatus, agent?.status, samples.length)
    };
  });
}

function nodeTone(status: string, agentStatus?: string, sampleCount = 0): NodeInsight["tone"] {
  if (["failed", "offline", "disabled", "stopped"].some((term) => status.includes(term)) || agentStatus === "offline") {
    return "danger";
  }
  if (["degraded", "maintenance", "pending"].some((term) => status.includes(term)) || agentStatus === "degraded") {
    return "warning";
  }
  if (["running", "active", "healthy", "ok"].some((term) => status.includes(term)) || sampleCount > 0) {
    return "ok";
  }
  return "muted";
}

function isSubscriptionReady(insight: NodeInsight): boolean {
  return insight.tone === "ok" && !["failed", "offline", "disabled"].includes(insight.node.status);
}

function toMbps(value: number): string {
  return (Math.round(((Number(value) || 0) * 8) / 100_000) / 10).toFixed(1);
}

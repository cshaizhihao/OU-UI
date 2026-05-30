import { useEffect, useMemo, useState } from "react";
import type { DashboardDTO } from "../api";
import type { NodeTraffic } from "../api";
import type { Agent } from "../data";
import { AnalyticsPanel } from "./Charts";
import { formatBytes, KpiGrid, MiniTable, SectionHeader, StatusTag, useFormatTime, useLocale, ViewHeading } from "./ConsolePrimitives";

type TrafficAuditWorkspaceProps = {
  agents: Agent[];
  data: DashboardDTO | null;
};

export function TrafficAuditWorkspace({ agents, data }: TrafficAuditWorkspaceProps) {
  const language = useLocale();
  const formatTime = useFormatTime();
  const traffic = data?.control.traffic ?? [];
  const nodeSummaries = useMemo(() => summarizeNodes(traffic), [traffic]);
  const [selectedNode, setSelectedNode] = useState("all");

  useEffect(() => {
    if (selectedNode === "all") {
      return;
    }
    if (!nodeSummaries.some((node) => node.nodeId === selectedNode)) {
      setSelectedNode(nodeSummaries[0]?.nodeId ?? "all");
    }
  }, [nodeSummaries, selectedNode]);

  const selectedTraffic = selectedNode === "all" ? traffic : traffic.filter((sample) => sample.nodeId === selectedNode);
  const selectedSummary =
    selectedNode === "all"
      ? summarizeSelection(traffic, language === "zh" ? "全部节点" : "All nodes", language === "zh" ? "混合" : "Mixed")
      : (nodeSummaries.find((node) => node.nodeId === selectedNode) ?? summarizeSelection(selectedTraffic, selectedNode, ""));

  return (
    <div className="workspace-view">
      <ViewHeading
        description="按每个独立生成节点追踪上传、下载、速率和连接数，便于租户配额与异常定位。"
        eyebrow="流量审计"
        title="单节点流量大屏"
      />

      <KpiGrid
        items={[
          { label: "采样节点", value: selectedNode === "all" ? String(nodeSummaries.length) : "1", delta: "独立节点口径" },
          { label: "上传累计", value: formatBytes(selectedSummary.rxBytes), delta: `${toMbps(selectedSummary.rxRateBps)} Mbps` },
          { label: "下载累计", value: formatBytes(selectedSummary.txBytes), delta: `${toMbps(selectedSummary.txRateBps)} Mbps` },
          {
            label: "当前连接",
            value: String(selectedSummary.connections),
            delta: language === "zh" ? `${selectedSummary.sampleCount} 个样本` : `${selectedSummary.sampleCount} samples`
          }
        ]}
      />

      <section className="panel node-audit-panel">
        <SectionHeader eyebrow="单节点选择" title="独立节点口径" />
        <div className="node-audit-layout">
          <div className="node-selector-list" role="listbox" aria-label={language === "zh" ? "单节点流量选择" : "Single-node traffic selector"}>
            <button
              aria-selected={selectedNode === "all"}
              className={selectedNode === "all" ? "selected" : ""}
              onClick={() => setSelectedNode("all")}
              type="button"
            >
              <strong>{language === "zh" ? "全部节点" : "All nodes"}</strong>
              <span>
                {language === "zh"
                  ? `${nodeSummaries.length} 个节点 / ${traffic.length} 个样本`
                  : `${nodeSummaries.length} nodes / ${traffic.length} samples`}
              </span>
            </button>
            {nodeSummaries.map((node) => (
              <button
                aria-selected={selectedNode === node.nodeId}
                className={selectedNode === node.nodeId ? "selected" : ""}
                key={node.nodeId}
                onClick={() => setSelectedNode(node.nodeId)}
                type="button"
              >
                <strong>{node.nodeId}</strong>
                <span>
                  {node.agentId || (language === "zh" ? "未绑定 Agent" : "Unassigned Agent")} / {toMbps(node.rxRateBps + node.txRateBps)} Mbps
                </span>
              </button>
            ))}
            {nodeSummaries.length === 0 ? <p className="empty-state">暂无流量样本</p> : null}
          </div>
          <div className="node-focus-card">
            <div>
              <span>当前节点</span>
              <strong>{selectedSummary.nodeId}</strong>
            </div>
            <div>
              <span>Agent</span>
              <strong>{selectedSummary.agentId || "未绑定 Agent"}</strong>
            </div>
            <div>
              <span>上行速率</span>
              <strong>{toMbps(selectedSummary.rxRateBps)} Mbps</strong>
            </div>
            <div>
              <span>下行速率</span>
              <strong>{toMbps(selectedSummary.txRateBps)} Mbps</strong>
            </div>
            <div>
              <span>最近采样</span>
              <strong>{formatTime(selectedSummary.collectedAt)}</strong>
            </div>
            <StatusTag tone={selectedSummary.sampleCount > 0 ? "ok" : "muted"}>{selectedSummary.sampleCount > 0 ? "在线" : "暂无记录"}</StatusTag>
          </div>
        </div>
      </section>

      <AnalyticsPanel agents={agents} selectedNodeId={selectedNode === "all" ? undefined : selectedNode} traffic={selectedTraffic} />

      <section className="panel">
        <SectionHeader eyebrow="审计明细" title="最近单节点样本" />
        <MiniTable
          columns={["节点", "Agent", "上传", "下载", "速率", "连接", "采集时间"]}
          rows={selectedTraffic.slice(0, 12).map((sample) => [
            sample.nodeId,
            sample.agentId,
            formatBytes(sample.rxBytes),
            formatBytes(sample.txBytes),
            `${toMbps(sample.rxRateBps + sample.txRateBps)} Mbps`,
            String(sample.connections),
            formatTime(sample.collectedAt)
          ])}
        />
      </section>
    </div>
  );
}

type NodeTrafficSummary = NodeTraffic & {
  sampleCount: number;
};

function summarizeNodes(traffic: NodeTraffic[]): NodeTrafficSummary[] {
  const nodes = new Map<string, NodeTraffic[]>();
  for (const sample of traffic) {
    const bucket = nodes.get(sample.nodeId) ?? [];
    bucket.push(sample);
    nodes.set(sample.nodeId, bucket);
  }
  return Array.from(nodes.entries())
    .map(([nodeId, samples]) => summarizeSelection(samples, nodeId, samples[0]?.agentId ?? ""))
    .sort((a, b) => new Date(b.collectedAt).getTime() - new Date(a.collectedAt).getTime());
}

function summarizeSelection(traffic: NodeTraffic[], nodeId: string, agentId: string): NodeTrafficSummary {
  const latest = [...traffic].sort((a, b) => new Date(b.collectedAt).getTime() - new Date(a.collectedAt).getTime())[0];
  return {
    nodeId,
    agentId: latest?.agentId ?? agentId,
    rxBytes: traffic.reduce((sum, item) => sum + item.rxBytes, 0),
    txBytes: traffic.reduce((sum, item) => sum + item.txBytes, 0),
    rxRateBps: latest?.rxRateBps ?? 0,
    txRateBps: latest?.txRateBps ?? 0,
    connections: latest?.connections ?? 0,
    collectedAt: latest?.collectedAt ?? "",
    sampleCount: traffic.length
  };
}

function toMbps(value: number): string {
  return (Math.round(((Number(value) || 0) * 8) / 100_000) / 10).toFixed(1);
}

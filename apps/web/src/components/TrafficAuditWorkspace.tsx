import { useEffect, useMemo, useState } from "react";
import { loadNodeTrafficSamples, type DashboardDTO, type ManagedNode, type NodeTraffic } from "../api";
import type { Agent } from "../data";
import { AnalyticsPanel } from "./Charts";
import { formatBytes, KpiGrid, MiniTable, SectionHeader, StatusTag, useFormatTime, useLocale, ViewHeading } from "./ConsolePrimitives";

type TrafficAuditWorkspaceProps = {
  agents: Agent[];
  data: DashboardDTO | null;
};

const trafficHistoryLimit = 96;

export function TrafficAuditWorkspace({ agents, data }: TrafficAuditWorkspaceProps) {
  const language = useLocale();
  const isZh = language === "zh-CN";
  const formatTime = useFormatTime();
  const traffic = data?.control.traffic ?? [];
  const managedNodes = data?.control.nodes ?? [];
  const nodeSummaries = useMemo(() => summarizeNodes(traffic, managedNodes), [traffic, managedNodes]);
  const [selectedNode, setSelectedNode] = useState("all");
  const [historyByNode, setHistoryByNode] = useState<Record<string, NodeTraffic[]>>({});
  const [loadingNode, setLoadingNode] = useState("");
  const [historyError, setHistoryError] = useState("");

  useEffect(() => {
    if (selectedNode === "all") {
      return;
    }
    if (!nodeSummaries.some((node) => node.nodeId === selectedNode)) {
      setSelectedNode(nodeSummaries[0]?.nodeId ?? "all");
    }
  }, [nodeSummaries, selectedNode]);

  useEffect(() => {
    if (selectedNode === "all" || historyByNode[selectedNode]) {
      return;
    }
    let cancelled = false;
    setLoadingNode(selectedNode);
    setHistoryError("");
    loadNodeTrafficSamples(selectedNode, trafficHistoryLimit)
      .then((samples) => {
        if (cancelled) {
          return;
        }
        setHistoryByNode((current) => ({ ...current, [selectedNode]: samples }));
      })
      .catch((err) => {
        if (cancelled) {
          return;
        }
        setHistoryError(err instanceof Error ? err.message : isZh ? "历史样本加载失败" : "Failed to load history");
      })
      .finally(() => {
        if (!cancelled) {
          setLoadingNode("");
        }
      });
    return () => {
      cancelled = true;
    };
  }, [historyByNode, isZh, selectedNode]);

  const latestForSelected = selectedNode === "all" ? traffic : traffic.filter((sample) => sample.nodeId === selectedNode);
  const selectedTraffic =
    selectedNode === "all" ? sortSamples(traffic).slice(-24) : sortSamples(historyByNode[selectedNode] ?? latestForSelected).slice(-trafficHistoryLimit);
  const selectedSummary =
    selectedNode === "all"
      ? summarizeAggregateSelection(traffic, isZh ? "全部节点" : "All nodes", isZh ? "混合" : "Mixed")
      : summarizeSelection(
          selectedTraffic,
          selectedNode,
          nodeSummaries.find((node) => node.nodeId === selectedNode)?.agentId ?? "",
          nodeSummaries.find((node) => node.nodeId === selectedNode)?.name,
          nodeSummaries.find((node) => node.nodeId === selectedNode)?.status
        );
  const lastUpdated = selectedSummary.collectedAt ? formatTime(selectedSummary.collectedAt) : isZh ? "暂无采样" : "No sample";
  const totalRateMbps = toMbpsNumber(selectedSummary.rxRateBps + selectedSummary.txRateBps);
  const peakRateMbps = peakRate(selectedTraffic);
  const sampleDelta =
    selectedNode === "all"
      ? isZh
        ? `${traffic.length} 条最新样本`
        : `${traffic.length} latest samples`
      : isZh
        ? `${selectedTraffic.length} 条历史样本`
        : `${selectedTraffic.length} history samples`;
  const tableRows = [...selectedTraffic]
    .sort((a, b) => new Date(b.collectedAt).getTime() - new Date(a.collectedAt).getTime())
    .slice(0, 12);

  return (
    <div className="workspace-view traffic-audit-workspace">
      <ViewHeading
        description={
          isZh
            ? "按每个托管节点追踪上传、下载、速率、连接数与最近采样，帮助定位配额压力和异常峰值。"
            : "Track upload, download, rate, connections, and latest samples for every managed node."
        }
        eyebrow={isZh ? "流量审计" : "Traffic audit"}
        title={isZh ? "单节点流量大屏" : "Per-node traffic dashboard"}
      />

      <KpiGrid
        items={[
          {
            label: isZh ? "采样节点" : "Sampled nodes",
            value: selectedNode === "all" ? String(nodeSummaries.length) : "1",
            delta: sampleDelta
          },
          { label: isZh ? "当前总速率" : "Current rate", value: `${totalRateMbps.toFixed(1)} Mbps`, delta: isZh ? "上下行合计" : "Upload + download" },
          { label: isZh ? "窗口峰值" : "Window peak", value: `${peakRateMbps.toFixed(1)} Mbps`, delta: isZh ? "最近样本峰值" : "Recent sample peak" },
          {
            label: isZh ? "当前连接" : "Connections",
            value: String(selectedSummary.connections),
            delta: lastUpdated
          }
        ]}
      />

      <section className="panel node-audit-panel">
        <SectionHeader
          eyebrow={isZh ? "单节点选择" : "Single-node selector"}
          title={isZh ? "节点、采样与健康状态" : "Node, samples, and health"}
        />
        <div className="node-audit-layout">
          <div className="node-selector-list" role="listbox" aria-label={isZh ? "单节点流量选择" : "Single-node traffic selector"}>
            <button
              aria-selected={selectedNode === "all"}
              className={selectedNode === "all" ? "selected" : ""}
              onClick={() => setSelectedNode("all")}
              type="button"
            >
              <strong>{isZh ? "全部节点" : "All nodes"}</strong>
              <span>{isZh ? `${nodeSummaries.length} 个节点 / ${traffic.length} 个样本` : `${nodeSummaries.length} nodes / ${traffic.length} samples`}</span>
            </button>
            {nodeSummaries.map((node) => (
              <button
                aria-selected={selectedNode === node.nodeId}
                className={selectedNode === node.nodeId ? "selected" : ""}
                key={node.nodeId}
                onClick={() => setSelectedNode(node.nodeId)}
                type="button"
              >
                <strong>{node.name || node.nodeId}</strong>
                <span>
                  {node.agentId || (isZh ? "未绑定 Agent" : "Unassigned Agent")} / {toMbps(node.rxRateBps + node.txRateBps)} Mbps
                </span>
              </button>
            ))}
            {nodeSummaries.length === 0 ? <p className="empty-state">{isZh ? "暂无流量样本" : "No traffic samples"}</p> : null}
          </div>
          <div className="node-focus-card">
            <div>
              <span>{isZh ? "当前节点" : "Current node"}</span>
              <strong>{selectedSummary.name || selectedSummary.nodeId}</strong>
            </div>
            <div>
              <span>Agent</span>
              <strong>{selectedSummary.agentId || (isZh ? "未绑定 Agent" : "Unassigned Agent")}</strong>
            </div>
            <div>
              <span>{isZh ? "上传速率" : "Upload rate"}</span>
              <strong>{toMbps(selectedSummary.rxRateBps)} Mbps</strong>
            </div>
            <div>
              <span>{isZh ? "下载速率" : "Download rate"}</span>
              <strong>{toMbps(selectedSummary.txRateBps)} Mbps</strong>
            </div>
            <div>
              <span>{isZh ? "最近采样" : "Latest sample"}</span>
              <strong>{lastUpdated}</strong>
            </div>
            <StatusTag tone={statusTone(selectedSummary, loadingNode === selectedNode)}>
              {loadingNode === selectedNode
                ? isZh
                  ? "加载中"
                  : "Loading"
                : selectedSummary.sampleCount > 0
                  ? isZh
                    ? "有采样"
                    : "Sampled"
                  : isZh
                    ? "暂无记录"
                    : "No records"}
            </StatusTag>
          </div>
        </div>
        {historyError ? <div className="notice-row notice-danger compact">{historyError}</div> : null}
      </section>

      <AnalyticsPanel
        agents={agents}
        isLoading={loadingNode === selectedNode}
        sampleWindowLabel={selectedNode === "all" ? (isZh ? "最新快照" : "Latest snapshot") : `${selectedTraffic.length} / ${trafficHistoryLimit}`}
        selectedNodeId={selectedNode === "all" ? undefined : selectedNode}
        traffic={selectedTraffic}
      />

      <section className="panel">
        <SectionHeader eyebrow={isZh ? "审计明细" : "Audit details"} title={isZh ? "最近单节点样本" : "Recent per-node samples"} />
        <MiniTable
          columns={[
            isZh ? "节点" : "Node",
            "Agent",
            isZh ? "上传" : "Upload",
            isZh ? "下载" : "Download",
            isZh ? "速率" : "Rate",
            isZh ? "连接" : "Connections",
            isZh ? "采集时间" : "Collected at"
          ]}
          emptyLabel={isZh ? "暂无流量样本" : "No traffic samples"}
          rows={tableRows.map((sample) => [
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
  name?: string;
  sampleCount: number;
  status?: string;
};

function summarizeNodes(traffic: NodeTraffic[], managedNodes: ManagedNode[]): NodeTrafficSummary[] {
  const nodes = new Map<string, NodeTraffic[]>();
  for (const sample of traffic) {
    const bucket = nodes.get(sample.nodeId) ?? [];
    bucket.push(sample);
    nodes.set(sample.nodeId, bucket);
  }
  const managed = managedNodes.map((node) => summarizeSelection(nodes.get(node.id) ?? [], node.id, node.agentId, node.name, node.status));
  const managedIDs = new Set(managedNodes.map((node) => node.id));
  const external = Array.from(nodes.entries())
    .filter(([nodeId]) => !managedIDs.has(nodeId))
    .map(([nodeId, samples]) => summarizeSelection(samples, nodeId, samples[0]?.agentId ?? ""));
  return [...managed, ...external].sort((a, b) => {
    if (a.sampleCount !== b.sampleCount) {
      return b.sampleCount - a.sampleCount;
    }
    return new Date(b.collectedAt).getTime() - new Date(a.collectedAt).getTime();
  });
}

function summarizeSelection(traffic: NodeTraffic[], nodeId: string, agentId: string, name?: string, status?: string): NodeTrafficSummary {
  const latest = [...traffic].sort((a, b) => new Date(b.collectedAt).getTime() - new Date(a.collectedAt).getTime())[0];
  return {
    nodeId,
    agentId: latest?.agentId ?? agentId,
    name,
    status,
    rxBytes: traffic.reduce((sum, item) => sum + item.rxBytes, 0),
    txBytes: traffic.reduce((sum, item) => sum + item.txBytes, 0),
    rxRateBps: latest?.rxRateBps ?? 0,
    txRateBps: latest?.txRateBps ?? 0,
    connections: latest?.connections ?? 0,
    collectedAt: latest?.collectedAt ?? "",
    sampleCount: traffic.length
  };
}

function summarizeAggregateSelection(traffic: NodeTraffic[], nodeId: string, agentId: string): NodeTrafficSummary {
  const latestByNode = new Map<string, NodeTraffic>();
  for (const sample of traffic) {
    const current = latestByNode.get(sample.nodeId);
    if (!current || new Date(sample.collectedAt).getTime() > new Date(current.collectedAt).getTime()) {
      latestByNode.set(sample.nodeId, sample);
    }
  }
  const latestSamples = Array.from(latestByNode.values());
  const latest = latestSamples.sort((a, b) => new Date(b.collectedAt).getTime() - new Date(a.collectedAt).getTime())[0];
  return {
    nodeId,
    agentId,
    rxBytes: traffic.reduce((sum, item) => sum + item.rxBytes, 0),
    txBytes: traffic.reduce((sum, item) => sum + item.txBytes, 0),
    rxRateBps: latestSamples.reduce((sum, item) => sum + item.rxRateBps, 0),
    txRateBps: latestSamples.reduce((sum, item) => sum + item.txRateBps, 0),
    connections: latestSamples.reduce((sum, item) => sum + item.connections, 0),
    collectedAt: latest?.collectedAt ?? "",
    sampleCount: traffic.length
  };
}

function sortSamples(samples: NodeTraffic[]): NodeTraffic[] {
  return [...samples].sort((a, b) => new Date(a.collectedAt).getTime() - new Date(b.collectedAt).getTime());
}

function peakRate(samples: NodeTraffic[]): number {
  return samples.reduce((peak, sample) => Math.max(peak, toMbpsNumber(sample.rxRateBps + sample.txRateBps)), 0);
}

function statusTone(summary: NodeTrafficSummary, loading: boolean): "muted" | "ok" | "warning" {
  if (loading) {
    return "warning";
  }
  return summary.sampleCount > 0 ? "ok" : "muted";
}

function toMbps(value: number): string {
  return toMbpsNumber(value).toFixed(1);
}

function toMbpsNumber(value: number): number {
  return Math.round(((Number(value) || 0) * 8) / 100_000) / 10;
}

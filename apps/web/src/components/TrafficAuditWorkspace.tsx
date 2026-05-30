import type { DashboardDTO } from "../api";
import type { Agent } from "../data";
import { AnalyticsPanel } from "./Charts";
import { formatBytes, formatTime, KpiGrid, MiniTable, SectionHeader, ViewHeading } from "./ConsolePrimitives";

type TrafficAuditWorkspaceProps = {
  agents: Agent[];
  data: DashboardDTO | null;
};

export function TrafficAuditWorkspace({ agents, data }: TrafficAuditWorkspaceProps) {
  const traffic = data?.control.traffic ?? [];
  const sources = new Set(traffic.map((item) => item.nodeId)).size;
  const rxBytes = traffic.reduce((sum, item) => sum + item.rxBytes, 0);
  const txBytes = traffic.reduce((sum, item) => sum + item.txBytes, 0);
  const rxRate = traffic.reduce((sum, item) => sum + item.rxRateBps, 0);
  const txRate = traffic.reduce((sum, item) => sum + item.txRateBps, 0);
  const connections = traffic.reduce((sum, item) => sum + item.connections, 0);

  return (
    <div className="workspace-view">
      <ViewHeading
        description="按每个独立生成节点追踪上传、下载、速率和连接数，便于租户配额与异常定位。"
        eyebrow="流量审计"
        title="单节点流量大屏"
      />

      <KpiGrid
        items={[
          { label: "采样节点", value: String(sources), delta: "独立节点口径" },
          { label: "上传累计", value: formatBytes(rxBytes), delta: `${toMbps(rxRate)} Mbps` },
          { label: "下载累计", value: formatBytes(txBytes), delta: `${toMbps(txRate)} Mbps` },
          { label: "连接数", value: String(connections), delta: "最新采样汇总" }
        ]}
      />

      <AnalyticsPanel agents={agents} traffic={traffic} />

      <section className="panel">
        <SectionHeader eyebrow="审计明细" title="最近单节点样本" />
        <MiniTable
          columns={["节点", "Agent", "上传", "下载", "速率", "连接", "采集时间"]}
          rows={traffic.slice(0, 12).map((sample) => [
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

function toMbps(value: number): string {
  return (Math.round(((Number(value) || 0) * 8) / 100_000) / 10).toFixed(1);
}

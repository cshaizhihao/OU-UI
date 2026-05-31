import * as echarts from "echarts/core";
import {
  GridComponent,
  LegendComponent,
  TooltipComponent,
  type GridComponentOption,
  type LegendComponentOption,
  type TooltipComponentOption
} from "echarts/components";
import { LineChart, type LineSeriesOption } from "echarts/charts";
import { CanvasRenderer } from "echarts/renderers";
import { useEffect, useMemo, useRef } from "react";
import type { NodeTraffic } from "../api";
import type { Agent } from "../data";
import { useLocale } from "./ConsolePrimitives";

echarts.use([GridComponent, LegendComponent, TooltipComponent, LineChart, CanvasRenderer]);

type ChartOption = echarts.ComposeOption<GridComponentOption | LegendComponentOption | TooltipComponentOption | LineSeriesOption>;

type AnalyticsPanelProps = {
  agents: Agent[];
  isLoading?: boolean;
  sampleWindowLabel?: string;
  selectedNodeId?: string;
  traffic: NodeTraffic[];
};

export function AnalyticsPanel({ agents, isLoading = false, sampleWindowLabel, selectedNodeId, traffic }: AnalyticsPanelProps) {
  const language = useLocale();
  const isZh = language === "zh-CN";
  const chartRef = useRef<HTMLDivElement>(null);
  const series = useMemo(() => buildTrafficSeries(agents, traffic), [agents, traffic]);
  const latestNodeSamples = useMemo(() => latestSamplesByNode(traffic), [traffic]);
  const peak = Math.max(...series.rxValues, ...series.txValues, 1);
  const combinedPeak = Math.max(...series.rxValues.map((rx, index) => rx + (series.txValues[index] ?? 0)), 1);
  const peakConnections = Math.max(...series.connections, 1);
  const activeNodes = selectedNodeId ? 1 : new Set(traffic.map((item) => item.nodeId)).size || agents.length;
  const totalMbps = selectedNodeId
    ? (series.rxValues.at(-1) ?? 0) + (series.txValues.at(-1) ?? 0)
    : latestNodeSamples.reduce((sum, item) => sum + bytesPerSecondToMbps(item.rxRateBps + item.txRateBps), 0);
  const currentConnections = selectedNodeId
    ? (series.connections.at(-1) ?? traffic.at(0)?.connections ?? 0)
    : latestNodeSamples.reduce((sum, item) => sum + item.connections, 0);
  const sampleCount = traffic.length || agents.length;

  useEffect(() => {
    if (!chartRef.current) {
      return;
    }
    const chart = echarts.init(chartRef.current, undefined, { renderer: "canvas" });
    const option: ChartOption = {
      color: ["#0f766e", "#2563eb", "#f59e0b"],
      grid: { left: 8, right: 14, top: 48, bottom: 20, containLabel: true },
      legend: {
        top: 6,
        right: 8,
        itemWidth: 18,
        itemHeight: 8,
        textStyle: { color: "#64748b", fontWeight: 800 }
      },
      tooltip: {
        trigger: "axis",
        backgroundColor: "rgba(15, 23, 42, 0.94)",
        borderColor: "rgba(148, 163, 184, 0.24)",
        textStyle: { color: "#f8fafc", fontWeight: 700 },
        valueFormatter: (value) => `${Number(value).toFixed(1)}`
      },
      xAxis: {
        type: "category",
        boundaryGap: false,
        data: series.labels,
        axisLine: { lineStyle: { color: "rgba(100, 116, 139, 0.3)" } },
        axisTick: { show: false },
        axisLabel: { color: "#64748b", fontWeight: 700, hideOverlap: true }
      },
      yAxis: [
        {
          type: "value",
          min: 0,
          max: Math.ceil(peak * 1.22),
          name: "Mbps",
          nameTextStyle: { color: "#64748b", fontWeight: 800 },
          splitLine: { lineStyle: { color: "rgba(148, 163, 184, 0.18)" } },
          axisLabel: { color: "#64748b", formatter: "{value}" }
        },
        {
          type: "value",
          min: 0,
          max: Math.ceil(peakConnections * 1.28),
          name: isZh ? "连接" : "Conn",
          nameTextStyle: { color: "#64748b", fontWeight: 800 },
          splitLine: { show: false },
          axisLabel: { color: "#94a3b8", formatter: "{value}" }
        }
      ],
      series: [
        {
          name: isZh ? "上传速率" : "Upload rate",
          type: "line",
          smooth: 0.38,
          symbol: "circle",
          symbolSize: 6,
          showSymbol: false,
          lineStyle: { width: 3, color: "#0f766e" },
          areaStyle: {
            color: {
              type: "linear",
              x: 0,
              y: 0,
              x2: 0,
              y2: 1,
              colorStops: [
                { offset: 0, color: "rgba(15, 118, 110, 0.28)" },
                { offset: 1, color: "rgba(15, 118, 110, 0)" }
              ]
            }
          },
          data: series.rxValues
        },
        {
          name: isZh ? "下载速率" : "Download rate",
          type: "line",
          smooth: 0.38,
          symbol: "circle",
          symbolSize: 6,
          showSymbol: false,
          lineStyle: { width: 3, color: "#2563eb" },
          areaStyle: {
            color: {
              type: "linear",
              x: 0,
              y: 0,
              x2: 0,
              y2: 1,
              colorStops: [
                { offset: 0, color: "rgba(37, 99, 235, 0.2)" },
                { offset: 1, color: "rgba(37, 99, 235, 0)" }
              ]
            }
          },
          data: series.txValues
        },
        {
          name: isZh ? "连接压力" : "Connection pressure",
          type: "line",
          yAxisIndex: 1,
          smooth: 0.22,
          symbol: "none",
          lineStyle: { width: 2, color: "#f59e0b", type: "dashed" },
          data: series.connections
        }
      ]
    };
    chart.setOption(option);
    const onResize = () => chart.resize();
    window.addEventListener("resize", onResize);
    return () => {
      window.removeEventListener("resize", onResize);
      chart.dispose();
    };
  }, [isZh, language, peak, peakConnections, series]);

  return (
    <section className="panel chart-panel traffic-ops-panel" id="metrics">
      <div className="traffic-panel-top">
        <div>
          <p className="eyebrow">{isZh ? "可观测性" : "Observability"}</p>
          <h2>{selectedNodeId ? (isZh ? "单节点波形与连接压力" : "Single-node wave and connection pressure") : isZh ? "流量波形与运行压力" : "Traffic wave and runtime pressure"}</h2>
          <span>{isZh ? "上下行速率使用 Mbps；虚线表示连接数压力。" : "Upload and download are Mbps; the dashed line tracks connection pressure."}</span>
        </div>
        <div className="traffic-window-pill">
          <span>{isZh ? "样本窗口" : "Sample window"}</span>
          <strong>{sampleWindowLabel ?? (isZh ? "最近样本" : "Recent samples")}</strong>
        </div>
      </div>
      <div className="traffic-wave-grid">
        <div className="traffic-wave-card">
          {isLoading ? <div className="chart-loading-sheen" aria-hidden="true" /> : null}
          <div ref={chartRef} className="traffic-wave" aria-label={isZh ? "单节点流量波形图" : "Per-node traffic waveform"} />
        </div>
        <div className="signal-stack">
          <SignalMetric label={isZh ? "当前" : "Current"} value={`${totalMbps.toFixed(1)} Mbps`} />
          <SignalMetric label={isZh ? "峰值" : "Peak"} value={`${combinedPeak.toFixed(1)} Mbps`} />
          <SignalMetric label={isZh ? "连接" : "Connections"} value={String(currentConnections)} />
          <SignalMetric label={isZh ? "样本" : "Samples"} value={String(sampleCount)} />
          <SignalMetric label={isZh ? "来源" : "Sources"} value={String(activeNodes)} />
        </div>
      </div>
    </section>
  );
}

function SignalMetric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function buildTrafficSeries(agents: Agent[], traffic: NodeTraffic[]) {
  const samples = [...traffic]
    .sort((a, b) => new Date(a.collectedAt).getTime() - new Date(b.collectedAt).getTime())
    .slice(-96);
  if (samples.length > 0) {
    return {
      labels: samples.map((item) => shortTime(item.collectedAt)),
      rxValues: samples.map((item) => bytesPerSecondToMbps(item.rxRateBps)),
      txValues: samples.map((item) => bytesPerSecondToMbps(item.txRateBps)),
      connections: samples.map((item) => item.connections)
    };
  }
  const fallback = agents.slice(0, 12);
  return {
    labels: fallback.map((agent) => agent.name),
    rxValues: fallback.map((agent) => agent.uplinkMbps),
    txValues: fallback.map((agent) => agent.downlinkMbps),
    connections: fallback.map((agent) => agent.queue)
  };
}

function latestSamplesByNode(traffic: NodeTraffic[]): NodeTraffic[] {
  const latest = new Map<string, NodeTraffic>();
  for (const sample of traffic) {
    const current = latest.get(sample.nodeId);
    if (!current || new Date(sample.collectedAt).getTime() > new Date(current.collectedAt).getTime()) {
      latest.set(sample.nodeId, sample);
    }
  }
  return Array.from(latest.values());
}

function bytesPerSecondToMbps(value: number): number {
  return Math.round(((Number(value) || 0) * 8) / 100_000) / 10;
}

function shortTime(value: string): string {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) {
    return value || "--";
  }
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

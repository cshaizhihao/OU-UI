import * as echarts from "echarts/core";
import { GridComponent, TooltipComponent, type GridComponentOption, type TooltipComponentOption } from "echarts/components";
import { LineChart, type LineSeriesOption } from "echarts/charts";
import { CanvasRenderer } from "echarts/renderers";
import { useEffect, useMemo, useRef } from "react";
import type { NodeTraffic } from "../api";
import type { Agent } from "../data";
import { useLocale } from "./ConsolePrimitives";

echarts.use([GridComponent, TooltipComponent, LineChart, CanvasRenderer]);

type ChartOption = echarts.ComposeOption<GridComponentOption | TooltipComponentOption | LineSeriesOption>;

type AnalyticsPanelProps = {
  agents: Agent[];
  traffic: NodeTraffic[];
};

export function AnalyticsPanel({ agents, traffic }: AnalyticsPanelProps) {
  const language = useLocale();
  const chartRef = useRef<HTMLDivElement>(null);
  const series = useMemo(() => buildTrafficSeries(agents, traffic), [agents, traffic]);
  const peak = Math.max(...series.values, 1);
  const activeNodes = new Set(traffic.map((item) => item.nodeId)).size || agents.length;
  const totalMbps = series.values.at(-1) ?? 0;

  useEffect(() => {
    if (!chartRef.current) {
      return;
    }
    const chart = echarts.init(chartRef.current, undefined, { renderer: "canvas" });
    const option: ChartOption = {
      grid: { left: 10, right: 16, top: 18, bottom: 18, containLabel: true },
      tooltip: {
        trigger: "axis",
        backgroundColor: "rgba(15, 23, 42, 0.92)",
        borderColor: "rgba(148, 163, 184, 0.24)",
        textStyle: { color: "#f8fafc", fontWeight: 700 },
        valueFormatter: (value) => `${Number(value).toFixed(1)} Mbps`
      },
      xAxis: {
        type: "category",
        boundaryGap: false,
        data: series.labels,
        axisLine: { lineStyle: { color: "rgba(100, 116, 139, 0.28)" } },
        axisTick: { show: false },
        axisLabel: { color: "#64748b", fontWeight: 700 }
      },
      yAxis: {
        type: "value",
        min: 0,
        max: Math.ceil(peak * 1.22),
        splitLine: { lineStyle: { color: "rgba(148, 163, 184, 0.18)" } },
        axisLabel: { color: "#64748b", formatter: "{value}" }
      },
      series: [
        {
          name: language === "zh" ? "吞吐量" : "Throughput",
          type: "line",
          smooth: 0.42,
          symbol: "circle",
          symbolSize: 7,
          showSymbol: false,
          lineStyle: {
            width: 4,
            color: {
              type: "linear",
              x: 0,
              y: 0,
              x2: 1,
              y2: 0,
              colorStops: [
                { offset: 0, color: "#14b8a6" },
                { offset: 0.55, color: "#38bdf8" },
                { offset: 1, color: "#f43f5e" }
              ]
            }
          },
          areaStyle: {
            color: {
              type: "linear",
              x: 0,
              y: 0,
              x2: 0,
              y2: 1,
              colorStops: [
                { offset: 0, color: "rgba(20, 184, 166, 0.32)" },
                { offset: 0.48, color: "rgba(56, 189, 248, 0.14)" },
                { offset: 1, color: "rgba(244, 63, 94, 0)" }
              ]
            }
          },
          data: series.values
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
  }, [peak, series]);

  return (
    <section className="panel chart-panel" id="metrics">
      <div className="section-heading">
        <div>
          <p className="eyebrow">{language === "zh" ? "可观测性" : "Observability"}</p>
          <h2>{language === "zh" ? "流量波形与运行压力" : "Traffic wave and runtime pressure"}</h2>
        </div>
        <select aria-label="Time range">
          <option>{language === "zh" ? "最近 12 个样本" : "Last 12 samples"}</option>
          <option>{language === "zh" ? "最近 24 个样本" : "Last 24 samples"}</option>
          <option>{language === "zh" ? "最近 7 天" : "Last 7 days"}</option>
        </select>
      </div>
      <div className="traffic-wave-grid">
        <div className="traffic-wave-card">
          <div ref={chartRef} className="traffic-wave" aria-label="ECharts traffic waveform" />
        </div>
        <div className="signal-stack">
          <SignalMetric label={language === "zh" ? "当前" : "Current"} value={`${totalMbps.toFixed(1)} Mbps`} />
          <SignalMetric label={language === "zh" ? "峰值" : "Peak"} value={`${peak.toFixed(1)} Mbps`} />
          <SignalMetric label={language === "zh" ? "来源" : "Sources"} value={String(activeNodes)} />
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
  const samples = traffic.slice(0, 12).reverse();
  if (samples.length > 0) {
    return {
      labels: samples.map((item) => shortTime(item.collectedAt)),
      values: samples.map((item) => bytesPerSecondToMbps(item.rxRateBps + item.txRateBps))
    };
  }
  const fallback = agents.slice(0, 12);
  return {
    labels: fallback.map((agent) => agent.name),
    values: fallback.map((agent) => agent.uplinkMbps + agent.downlinkMbps)
  };
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

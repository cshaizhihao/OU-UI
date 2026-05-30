import { createContext, useContext, type ReactNode } from "react";
import type { Locale } from "./Shell";

const LocaleContext = createContext<Locale>("zh");

export function LocaleProvider({ children, language }: { children: ReactNode; language: Locale }) {
  return <LocaleContext.Provider value={language}>{children}</LocaleContext.Provider>;
}

export function useLocale() {
  return useContext(LocaleContext);
}

export function useCopy<T extends Record<Locale, Record<string, string>>>(copy: T): T[Locale] {
  return copy[useLocale()];
}

export function translate(value: string, language: Locale): string {
  if (language === "zh") {
    return value;
  }
  return englishPhrases[value] ?? value;
}

const englishPhrases: Record<string, string> = {
  "Agent 与托管节点": "Agents and managed nodes",
  "Agent、节点、任务或租户": "Agent, node, task, or tenant",
  "Agent 监控": "Agent monitor",
  "Agent 明细": "Agent detail",
  "API Key 与 AI 运维": "API keys and AI operations",
  "Clash 托管": "Clash hosting",
  "GeoIP / GeoSite / 协议规则": "GeoIP / GeoSite / protocol rules",
  "RBAC 与配额隔离": "RBAC and quota isolation",
  "REST API": "REST API",
  "运行态势": "Operations posture",
  "单节点流量大屏": "Per-node traffic dashboard",
  "多租户": "Multi-tenant",
  "告警与订阅聚合": "Alerts and subscription aggregation",
  "告警投递": "Alert delivery",
  "告警事件": "Alert events",
  "外部订阅聚合": "External subscription aggregation",
  "工作区加载中": "Workspace loading",
  "已配置分流": "Configured routing",
  "当前入口选择": "Current entry decision",
  "托管文件": "Hosted files",
  "控制面状态": "Control-plane status",
  "新增分流规则": "New routing rule",
  "最新事件": "Latest events",
  "最近单节点样本": "Recent per-node samples",
  "最近投递状态": "Recent delivery status",
  "流量审计": "Traffic audit",
  "流量波形与运行压力": "Traffic wave and runtime pressure",
  "规则生成工作区": "Rule generation workspace",
  "规则源与分组": "Rule providers and groups",
  "负载均衡组": "Load-balancer groups",
  "路由分流": "Routing",
  "运行时服务控制": "Runtime service control",
  "运维控制台": "Operations console",
  "配额矩阵": "Quota matrix",
  "关键指标": "Key metrics",
  "节点下发": "Node dispatch",
  "节点健康": "Node health",
  "节点管理": "Node management",
  "候选节点": "Candidate nodes",
  "可观测性": "Observability",
  "开放集成": "Open integrations",
  "租户与用户": "Tenants and users",
  "统一入口": "Unified entry",
  "总览": "Overview",
  "自动化运维": "Ops automation",
  "上传累计": "Upload total",
  "下载累计": "Download total",
  "采样节点": "Sampled nodes",
  "连接数": "Connections",
  "在线 Agent": "Online Agents",
  "托管节点": "Managed nodes",
  "单节点流量": "Per-node traffic",
  "开放告警": "Open alerts",
  "健康节点": "Healthy nodes",
  "任务队列": "Task queue",
  "协议能力": "Protocol capability",
  "外部与自建节点": "External and self-built nodes",
  "订阅托管地址": "Hosted subscription URL",
  "第三方对接预留": "Third-party integration reserve",
  "故障排查问答": "Troubleshooting Q&A",
  "签发作用域密钥": "Issue scoped keys",
  "主机调优": "Host tuning",
  "控制面总览": "Control plane overview",
  "按每个独立生成节点追踪上传、下载、速率和连接数，便于租户配额与异常定位。": "Track upload, download, rate, and connections for each generated node.",
  "独立节点口径": "Per-node scope",
  "最新采样汇总": "Latest sample summary",
  "自建与外部节点统一管理": "Self-built and external nodes",
  "心跳在线": "Heartbeat online",
  "运行时状态": "Runtime status",
  "部署与调优任务": "Deployment and tuning tasks",
  "等待实时快照": "Waiting for live snapshot",
  "Webhook 自动投递": "Webhook automation"
};

export type StatItem = {
  label: string;
  value: string;
  delta?: string;
};

export function ViewHeading({
  eyebrow,
  title,
  description,
  actions
}: {
  actions?: ReactNode;
  description?: string;
  eyebrow: string;
  title: string;
}) {
  const language = useLocale();
  return (
    <div className="view-heading">
      <div>
        <p className="eyebrow">{translate(eyebrow, language)}</p>
        <h2>{translate(title, language)}</h2>
        {description ? <span>{translate(description, language)}</span> : null}
      </div>
      {actions ? <div className="button-row">{actions}</div> : null}
    </div>
  );
}

export function KpiGrid({ items }: { items: StatItem[] }) {
  const language = useLocale();
  return (
    <section className="kpi-grid" aria-label="关键指标">
      {items.map((item) => (
        <article className="kpi-card" key={item.label}>
          <span>{translate(item.label, language)}</span>
          <strong>{item.value}</strong>
          {item.delta ? <em>{translate(item.delta, language)}</em> : null}
        </article>
      ))}
    </section>
  );
}

export function SectionHeader({
  eyebrow,
  title,
  actions
}: {
  actions?: ReactNode;
  eyebrow?: string;
  title: string;
}) {
  const language = useLocale();
  return (
    <div className="section-heading">
      <div>
        {eyebrow ? <p className="eyebrow">{translate(eyebrow, language)}</p> : null}
        <h2>{translate(title, language)}</h2>
      </div>
      {actions}
    </div>
  );
}

export function NoticeRow({
  children,
  tone = "info"
}: {
  children: ReactNode;
  tone?: "info" | "danger" | "success";
}) {
  return <div className={`notice-row notice-${tone}`}>{children}</div>;
}

export function MiniTable({
  columns,
  emptyLabel = "暂无记录",
  rows
}: {
  columns: string[];
  emptyLabel?: string;
  rows: string[][];
}) {
  const language = useLocale();
  return (
    <div className="mini-table">
      <div className="mini-row mini-head" style={{ gridTemplateColumns: `repeat(${columns.length}, minmax(0, 1fr))` }}>
        {columns.map((column) => (
          <strong key={column}>{translate(column, language)}</strong>
        ))}
      </div>
      {rows.length ? (
        rows.map((row, index) => (
          <div
            className="mini-row"
            key={`${row.join("-")}-${index}`}
            style={{ gridTemplateColumns: `repeat(${columns.length}, minmax(0, 1fr))` }}
          >
            {row.map((cell, cellIndex) => (
              <span key={`${cell}-${cellIndex}`}>{cell ? translate(cell, language) : "-"}</span>
            ))}
          </div>
        ))
      ) : (
        <div className="mini-row empty">
          <span>{translate(emptyLabel, language)}</span>
        </div>
      )}
    </div>
  );
}

export function StatusTag({
  children,
  tone = "info"
}: {
  children: ReactNode;
  tone?: "danger" | "info" | "muted" | "ok" | "warning";
}) {
  return <span className={`status service-pill service-${tone}`}>{children}</span>;
}

export function parseCSV(value: string): string[] {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

export function parseLines(value: string): string[] {
  return value
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean);
}

export function stringsTrim(value: string): string {
  return value.trim();
}

export function gbToBytes(value: number): number {
  return Math.max(0, Math.round(Number(value) || 0)) * 1024 * 1024 * 1024;
}

export function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) {
    return "0 GB";
  }
  if (value >= 1024 * 1024 * 1024) {
    return `${Math.round(value / 1024 / 1024 / 1024)} GB`;
  }
  return `${Math.round(value / 1024 / 1024)} MB`;
}

export function formatTime(value?: string): string {
  if (!value) {
    return "从未";
  }
  const time = new Date(value).getTime();
  if (!Number.isFinite(time)) {
    return value;
  }
  const minutes = Math.max(0, Math.round((Date.now() - time) / 60000));
  if (minutes < 60) {
    return `${minutes} 分钟前`;
  }
  const hours = Math.round(minutes / 60);
  return hours < 48 ? `${hours} 小时前` : `${Math.round(hours / 24)} 天前`;
}

export function taskTone(status: string): "pending" | "running" | "success" | "failed" {
  if (status === "succeeded" || status === "success") {
    return "success";
  }
  if (status === "running") {
    return "running";
  }
  if (status === "failed" || status === "canceled") {
    return "failed";
  }
  return "pending";
}

export function formatServiceStatus(status: string): string {
  const normalized = status.replace(/[_-]+/g, " ").replace(/\s+/g, " ").trim();
  const labels: Record<string, string> = {
    active: "运行中",
    canceled: "已取消",
    deployed: "已部署",
    failed: "失败",
    offline: "离线",
    online: "在线",
    pending: "等待中",
    queued: "已排队",
    running: "执行中",
    stopped: "已停止",
    success: "成功",
    succeeded: "成功"
  };
  return labels[normalized.toLowerCase()] ?? normalized;
}

export function isAPIKeyResponse(value: unknown): value is { apiKey: string } {
  return Boolean(value && typeof value === "object" && "apiKey" in value);
}

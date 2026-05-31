import { createContext, useContext, useEffect, type ReactNode } from "react";
import type { Locale } from "./Shell";

const LocaleContext = createContext<Locale>("zh");

export function LocaleProvider({ children, language }: { children: ReactNode; language: Locale }) {
  useEffect(() => {
    if (language !== "en") {
      return;
    }
    const translateRoot = () => translateDOM(document.getElementById("root") ?? document.body);
    translateRoot();
    const observer = new MutationObserver(() => translateRoot());
    observer.observe(document.getElementById("root") ?? document.body, {
      attributes: true,
      childList: true,
      subtree: true
    });
    return () => observer.disconnect();
  }, [language]);

  return <LocaleContext.Provider value={language}>{children}</LocaleContext.Provider>;
}

export function useLocale() {
  return useContext(LocaleContext);
}

export function useCopy<T extends Record<Locale, Record<string, string>>>(copy: T): T[Locale] {
  return copy[useLocale()];
}

export function useT() {
  const language = useLocale();
  return (value: string) => translate(value, language);
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
  "节点详情与订阅可用性": "Node detail and subscription readiness",
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
  "Webhook 自动投递": "Webhook automation",
  "OU-UI 运维控制台": "OU-UI Operations Console",
  "节点运维控制台": "Node Ops Console",
  "聚合 Agent、节点、告警和任务状态": "Unified view of Agents, nodes, alerts, and tasks",
  "管理 Agent、托管节点、运行时服务和任务队列": "Manage Agents, generated nodes, runtime services, and task queues",
  "按独立节点查看上传、下载、速率和连接数": "Inspect upload, download, rate, and connections per node",
  "配置 GeoIP、GeoSite、广告过滤和协议分流": "Configure GeoIP, GeoSite, ad blocking, and protocol split rules",
  "按延迟、丢包和权重选择统一入口后端": "Select unified-entry backends by latency, packet loss, and weight",
  "管理 Webhook、Telegram、Server 酱和订阅聚合": "Manage webhooks, Telegram, ServerChan, and subscription aggregation",
  "维护 Rule Provider、Proxy Group 和托管 YAML": "Maintain Rule Providers, Proxy Groups, and hosted YAML",
  "配置租户、子账号、节点访问权和流量配额": "Configure tenants, sub-users, node access, and traffic quotas",
  "签发 API Key，并让 Copilot 分析异常与日志特征": "Issue API keys and let Copilot analyze incidents and log signals",
  "这里集中处理 Agent、生成节点、运行时部署和服务状态，不再和其他业务上下堆叠。": "This workspace focuses on Agents, generated nodes, runtime deployment, and service status.",
  "这里保留控制面全局态势，不再混放所有配置表单。": "This view keeps the global control-plane posture separate from configuration forms.",
  "用可视化规则直接生成 Xray 路由 payload，并通过任务队列下发到具备能力的 Agent。": "Generate Xray routing payloads visually and dispatch them to capable Agents.",
  "将多个 Agent 聚合为统一入口，根据延迟、丢包和权重动态选择后端。": "Aggregate Agents into a unified entry and select backends dynamically.",
  "把告警投递和外部订阅聚合放在同一个自动化运维工作区，避免散落在长页面。": "Keep alert delivery and external subscription aggregation in one automation workspace.",
  "独立管理 Rule Provider、Proxy Group、节点选择和完整 YAML 托管输出。": "Manage Rule Providers, Proxy Groups, node selection, and hosted YAML output.",
  "租户、子账号、节点访问权、月度配额和单节点配额在独立工作区内完成。": "Manage tenants, sub-users, node access, monthly quotas, and per-node quotas here.",
  "为第三方系统预留 REST API 接入，并通过 AI Copilot 汇总异常流量与错误日志。": "Reserve REST API integration for third-party systems and summarize incidents with AI Copilot.",
  "刷新快照": "Refresh snapshot",
  "暂无 Agent 数据": "No Agent data",
  "暂无任务": "No tasks",
  "最近执行队列": "Recent task queue",
  "任务": "Tasks",
  "告警": "Alerts",
  "事件": "Event",
  "来源": "Source",
  "状态": "Status",
  "时间": "Time",
  "已投递": "Delivered",
  "待处理": "Pending",
  "未绑定 Agent": "Unassigned Agent",
  "运行时": "Runtime",
  "协议": "Protocol",
  "队列策略": "Queue policy",
  "托管 reload，保持活动连接": "Managed reload, keep active sessions",
  "托管 restart，维护窗口执行": "Managed restart during maintenance",
  "外部服务，等待人工确认": "External service, wait for approval",
  "暂无可用 Agent": "No available Agent",
  "节点": "Node",
  "名称": "Name",
  "类型": "Type",
  "匹配内容": "Match",
  "动作": "Action",
  "保存规则": "Save rule",
  "规则编辑器": "Rule editor",
  "当前规则": "Current rules",
  "下发目标": "Dispatch targets",
  "在线 Agent 能力": "Online Agent capability",
  "一键 BBR v3": "One-click BBR v3",
  "下发到在线 Agent": "Apply to online Agents",
  "广告过滤": "Ad blocking",
  "域名": "Domain",
  "阻断": "Block",
  "直连": "Direct",
  "代理": "Proxy",
  "规则": "Rule",
  "匹配": "Match",
  "启用": "Enabled",
  "停用": "Disabled",
  "区域": "Region",
  "能力": "Capability",
  "未上报": "Not reported",
  "暂无可执行主机调优的 Agent": "No Agent is available for host tuning",
  "组名": "Group name",
  "入口标识": "Entry tag",
  "策略": "Strategy",
  "延迟 + 丢包": "Latency + loss",
  "权重优先": "Weighted",
  "探测间隔秒": "Check interval seconds",
  "创建 HA 组": "Create HA group",
  "均衡组": "Balancing group",
  "创建入口组": "Create entry group",
  "决策": "Decision",
  "组": "Group",
  "入口": "Entry",
  "选中后端": "Selected backend",
  "得分": "Score",
  "导入首个订阅": "Import first subscription",
  "请先创建一个订阅源": "Create a subscription source first",
  "Webhook 测试": "Webhook test",
  "Server 酱": "ServerChan",
  "事件类型": "Event types",
  "保存通道": "Save channel",
  "测试": "Test",
  "暂无 Webhook": "No webhooks",
  "订阅控制台": "Subscription console",
  "订阅源": "Subscription sources",
  "外部节点": "External nodes",
  "最近导入": "Latest import",
  "异常源": "Error sources",
  "聚合订阅预览": "Aggregate subscription preview",
  "Rule Provider 与 Proxy Group 直接可用": "Ready for Rule Providers and Proxy Groups",
  "兼容通用订阅客户端": "Compatible with common subscription clients",
  "调试和迁移时快速检查节点": "Quick node checks for debugging and migration",
  "预留 sing-box 客户端托管输出": "Reserved hosted output for sing-box clients",
  "设为输出格式": "Set output format",
  "复制地址": "Copy URL",
  "订阅地址已复制": "Subscription URL copied",
  "托管地址已复制": "Hosted URL copied",
  "内联内容": "Inline content",
  "添加订阅源": "Add source",
  "订阅节点": "Subscription nodes",
  "外部节点池": "External node pool",
  "聚合输出": "Aggregate output",
  "聚合订阅格式": "Aggregate subscription format",
  "生成聚合订阅": "Generate aggregate",
  "格式": "Format",
  "质量": "Quality",
  "级别": "Severity",
  "暂停": "Paused",
  "配置名称": "Profile name",
  "Provider 名称": "Provider name",
  "Provider 类型": "Provider type",
  "分组类型": "Group type",
  "分组节点": "Group nodes",
  "配置节点": "Profile nodes",
  "配置路径未上报": "Config path not reported",
  "生成 YAML": "Generate YAML",
  "生成 Clash 片段": "Generate Clash snippet",
  "已生成配置": "Generated profiles",
  "地址": "Address",
  "暂无 Clash 配置": "No Clash profiles",
  "作用域": "Scopes",
  "签发密钥": "Issue key",
  "问题": "Question",
  "询问 Copilot": "Ask Copilot",
  "资源": "Resource",
  "用途": "Purpose",
  "认证": "Auth",
  "OpenAPI 文档": "OpenAPI document",
  "签发集成密钥": "Issue integration key",
  "AI 运维问答": "AI operations Q&A",
  "聚合订阅": "Aggregate subscription",
  "治理操作": "Governance actions",
  "租户与子账号策略": "Tenant and sub-user policies",
  "租户治理": "Tenant governance",
  "子账号治理": "Sub-user governance",
  "保存租户策略": "Save tenant policy",
  "保存子账号策略": "Save sub-user policy",
  "临时密码": "Temporary password",
  "留空则不变": "Leave blank to keep unchanged",
  "请先选择一个租户": "Select a tenant first",
  "请先选择一个子账号": "Select a sub-user first",
  "租户总数": "Tenants",
  "子账号": "Sub-users",
  "配额压力": "Quota pressure",
  "实时连接占用": "Live connection usage",
  "租户运营台": "Tenant operations",
  "租户访问矩阵": "Tenant access matrix",
  "租户画像": "Tenant profile",
  "等待租户接入": "Waiting for tenant intake",
  "租户名称": "Tenant name",
  "节点访问": "Node access",
  "月度用量": "Monthly usage",
  "峰值单节点": "Peak node usage",
  "连接占用": "Connection usage",
  "账号运营": "Account operations",
  "子账号访问看板": "Sub-user access board",
  "创建租户后会显示配额、访问范围和子账号继承关系": "Create a tenant to inspect quotas, access scopes, and sub-user inheritance",
  "暂无租户": "No tenants",
  "暂无子账号": "No sub-users",
  "至少 10 位临时密码": "Temporary password with at least 10 characters",
  "单节点选择": "Single-node selector",
  "全部节点": "All nodes",
  "个节点": "nodes",
  "个样本": "samples",
  "当前节点": "Current node",
  "上行速率": "Upload rate",
  "下行速率": "Download rate",
  "最近采样": "Latest sample",
  "暂无流量样本": "No traffic samples",
  "当前连接": "Current connections",
  "上传速率": "Upload rate",
  "下载速率": "Download rate",
  "单节点波形与连接压力": "Single-node wave and connection pressure",
  "单节点流量波形图": "Per-node traffic waveform",
  "连接": "Connections",
  "月度 GB 配额": "Monthly GB quota",
  "单节点 GB 配额": "Per-node GB quota",
  "月度配额": "Monthly quota",
  "单节点配额": "Per-node quota",
  "最大连接数": "Max connections",
  "创建租户": "Create tenant",
  "用户名": "Username",
  "密码": "Password",
  "租户 ID": "Tenant ID",
  "创建子账号": "Create sub-user",
  "租户": "Tenant",
  "用户": "User",
  "角色": "Role",
  "主租户": "Root tenant",
  "健康": "Healthy",
  "未限额": "No quota",
  "接近配额": "Watch quota",
  "已触顶": "Limit reached",
  "上传": "Upload",
  "下载": "Download",
  "速率": "Rate",
  "采集时间": "Collected at",
  "审计明细": "Audit details",
  "托管节点选择": "Managed node selector",
  "等待节点接入": "Waiting for node intake",
  "Agent 上报托管节点后，这里会显示运行时、流量、订阅和服务状态。": "After Agents report managed nodes, runtime, traffic, subscription, and service state appear here.",
  "实时速率": "Live rate",
  "累计上传": "Upload total",
  "累计下载": "Download total",
  "订阅状态": "Subscription status",
  "可纳入聚合订阅": "Ready for aggregate subscription",
  "等待健康检查": "Waiting for health check",
  "Clash / V2Ray / Sing-box 可复用该节点": "Clash / V2Ray / Sing-box can reuse this node",
  "需确认服务可用、节点在线或有采样": "Needs healthy service, online Agent, or traffic samples",
  "最近变更": "Latest change",
  "无最近错误": "No recent error",
  "复制订阅标识": "Copy subscription ID",
  "查看流量样本": "View traffic samples",
  "输入密码": "Enter password",
  "保持登录": "Keep signed in",
  "重置密码": "Reset password",
  "登录中": "Signing in",
  "登录": "Sign in",
  "安全工作区": "Secure workspace",
  "管理 Agent、运行时、协议与自动化任务队列。": "Manage Agents, runtimes, protocols, and automation queues.",
  "5 类代理协议": "5 proxy protocols",
  "实时链路监控": "Live link monitoring",
  "重试": "Retry",
  "暂无记录": "No records",
  "指标加载中": "Metrics loading",
  "从未": "Never",
  "运行中": "Active",
  "已取消": "Canceled",
  "已部署": "Deployed",
  "失败": "Failed",
  "离线": "Offline",
  "在线": "Online",
  "等待中": "Pending",
  "已排队": "Queued",
  "执行中": "Running",
  "已停止": "Stopped",
  "计费系统集成": "Billing system integration",
  "为什么最新 Agent 处于 degraded 状态？": "Why is the newest Agent degraded?",
  "为第三方计费、发卡和运维系统预留 REST API 接入，并通过 AI Copilot 汇总异常流量与错误日志。": "Reserve REST API access for billing, activation, and operations systems, with AI Copilot summarizing traffic anomalies and error logs.",
  "密钥治理台": "API key governance",
  "只读 API": "Read-only API",
  "读写 API": "Read/write API",
  "控制面通配": "Control-plane wildcard",
  "查询租户、节点、流量和订阅": "Query tenants, nodes, traffic, and subscriptions",
  "允许创建任务、写入配置和集成回调": "Allow tasks, configuration writes, and integration callbacks",
  "保留给可信计费或运维系统": "Reserved for trusted billing or operations systems",
  "过期日期": "Expiration date",
  "主租户 / 全局": "Root tenant / global",
  "签发与治理集成密钥": "Issue and govern integration keys",
  "更新或吊销密钥": "Update or revoke keys",
  "暂无 API Key，签发后会在这里进行暂停、启用和吊销。": "No API keys yet. Issued keys can be paused, resumed, or revoked here.",
  "租户可访问所有 Agent 与托管节点": "Tenant can access all Agents and managed nodes",
  "按 Agent 或单节点精确授权": "Grant access by Agent or individual node",
  "等待 Agent 或托管节点接入": "Waiting for Agents or managed nodes",
  "长期有效": "No expiration",
  "最后使用": "Last used",
  "过期": "Expires",
  "吊销": "Revoke",
  "成功": "Success"
};

function translateDOM(root: ParentNode) {
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
    acceptNode(node) {
      const parent = node.parentElement;
      if (!parent || ["SCRIPT", "STYLE", "TEXTAREA", "INPUT", "OPTION"].includes(parent.tagName)) {
        return NodeFilter.FILTER_REJECT;
      }
      return NodeFilter.FILTER_ACCEPT;
    }
  });
  const nodes: Text[] = [];
  while (walker.nextNode()) {
    nodes.push(walker.currentNode as Text);
  }
  for (const node of nodes) {
    const raw = node.nodeValue ?? "";
    const trimmed = raw.trim();
    const translated = englishPhrases[trimmed];
    if (translated && translated !== trimmed) {
      node.nodeValue = raw.replace(trimmed, translated);
    }
  }
  if (root instanceof Element || root instanceof Document || root instanceof DocumentFragment) {
    const elements =
      root instanceof Element
        ? [root, ...Array.from(root.querySelectorAll("*"))]
        : Array.from(root.querySelectorAll("*"));
    for (const element of elements) {
      for (const attr of ["placeholder", "aria-label", "title"]) {
        const value = element.getAttribute(attr);
        if (value && englishPhrases[value]) {
          element.setAttribute(attr, englishPhrases[value]);
        }
      }
    }
  }
}

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
              <span key={`${cell}-${cellIndex}`} title={cell || undefined}>
                {cell ? translate(cell, language) : "-"}
              </span>
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
  return formatTimeForLocale(value, "zh");
}

export function useFormatTime() {
  const language = useLocale();
  return (value?: string) => formatTimeForLocale(value, language);
}

function formatTimeForLocale(value: string | undefined, language: Locale): string {
  if (!value) {
    return language === "zh" ? "从未" : "Never";
  }
  const time = new Date(value).getTime();
  if (!Number.isFinite(time)) {
    return value;
  }
  const minutes = Math.max(0, Math.round((Date.now() - time) / 60000));
  if (minutes < 60) {
    return language === "zh" ? `${minutes} 分钟前` : `${minutes} min ago`;
  }
  const hours = Math.round(minutes / 60);
  if (hours < 48) {
    return language === "zh" ? `${hours} 小时前` : `${hours} h ago`;
  }
  const days = Math.round(hours / 24);
  return language === "zh" ? `${days} 天前` : `${days} d ago`;
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

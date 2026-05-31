import { useMemo, useState, type FormEvent } from "react";
import {
  aggregateSubscriptionURL,
  createSubscription,
  createWebhook,
  importSubscription,
  loadAggregateSubscription,
  testWebhook,
  type AggregateSubscriptionFormat,
  type AlertEvent,
  type DashboardDTO,
  type WebhookEndpoint
} from "../api";
import {
  MiniTable,
  NoticeRow,
  parseCSV,
  SectionHeader,
  useFormatTime,
  useLocale,
  ViewHeading
} from "./ConsolePrimitives";

type OperationsWorkspaceProps = {
  data: DashboardDTO | null;
  disabled?: boolean;
  onRefresh?: () => void;
};

export function OperationsWorkspace({ data, disabled = false, onRefresh }: OperationsWorkspaceProps) {
  const control = data?.control;
  const [busy, setBusy] = useState("");
  const [message, setMessage] = useState("");
  const [webhook, setWebhook] = useState({
    name: "运维告警",
    kind: "generic",
    url: "",
    secret: "",
    chatId: "",
    eventTypes: "agent.offline,traffic.quota.exceeded"
  });
  const [subscription, setSubscription] = useState({
    name: "外部节点池",
    url: "",
    content: ""
  });
  const [aggregateFormat, setAggregateFormat] = useState<AggregateSubscriptionFormat>("clash");
  const [aggregateContent, setAggregateContent] = useState("");
  const language = useLocale();
  const formatTime = useFormatTime();
  const subscriptionSummary = useMemo(() => buildSubscriptionSummary(control), [control]);
  const controlsDisabled = disabled || !control;

  async function runAction(label: string, action: () => Promise<unknown>) {
    setBusy(label);
    setMessage("");
    try {
      await action();
      setMessage(`${label} 已完成`);
      await Promise.resolve(onRefresh?.());
    } catch (err) {
      setMessage(err instanceof Error ? err.message : `${label} 执行失败`);
    } finally {
      setBusy("");
    }
  }

  function handleCreateWebhook(event: FormEvent) {
    event.preventDefault();
    void runAction("告警 Webhook", () =>
      createWebhook({
        ...webhook,
        enabled: true,
        eventTypes: parseCSV(webhook.eventTypes)
      })
    );
  }

  function handleCreateSubscription(event: FormEvent) {
    event.preventDefault();
    void runAction("外部订阅", () =>
      createSubscription({
        name: subscription.name,
        url: subscription.url,
        content: subscription.content,
        format: "auto",
        enabled: true
      })
    );
  }

  function handleImportFirstSubscription() {
    const sub = control?.subscriptions[0];
    if (!sub) {
      setMessage("请先创建一个订阅源");
      return;
    }
    void runAction("订阅导入", () => importSubscription(sub.id));
  }

  function handleLoadAggregateSubscription() {
    void runAction("聚合订阅", async () => {
      const content = await loadAggregateSubscription(aggregateFormat);
      setAggregateContent(content);
      return content;
    });
  }

  async function handleCopyAggregateURL(format: AggregateSubscriptionFormat) {
    try {
      await navigator.clipboard.writeText(aggregateSubscriptionURL(format));
      setMessage("订阅地址已复制");
    } catch {
      setMessage(aggregateSubscriptionURL(format));
    }
  }

  return (
    <div className="workspace-view">
      <ViewHeading
        actions={
          <button className="ghost-button" disabled={Boolean(busy) || controlsDisabled} onClick={handleImportFirstSubscription} type="button">
            导入首个订阅
          </button>
        }
        description="把告警投递和外部订阅聚合放在同一个自动化运维工作区，避免散落在长页面。"
        eyebrow="自动化运维"
        title="告警与订阅聚合"
      />
      {message ? <NoticeRow>{message}</NoticeRow> : null}

      <section className="panel subscription-command-panel">
        <SectionHeader eyebrow="订阅控制台" title="聚合订阅预览" />
        <div className="subscription-command-grid">
          {subscriptionFormats.map((format) => (
            <article className={aggregateFormat === format.value ? "subscription-card selected" : "subscription-card"} key={format.value}>
              <div>
                <span>{format.label}</span>
                <strong>{format.description}</strong>
              </div>
              <code>{aggregateSubscriptionURL(format.value)}</code>
              <div className="subscription-card-meta">
                <span>{formatAvailableNodes(subscriptionSummary.enabledNodes, language)}</span>
                <span>{formatSourceCount(subscriptionSummary.activeSources, language)}</span>
                <span>{formatTime(subscriptionSummary.lastFetchedAt)}</span>
              </div>
              <div className="button-row">
                <button className="ghost-button" onClick={() => setAggregateFormat(format.value)} type="button">
                  设为输出格式
                </button>
                <button className="ghost-button" onClick={() => void handleCopyAggregateURL(format.value)} type="button">
                  复制地址
                </button>
              </div>
            </article>
          ))}
        </div>
        <div className="subscription-health-strip">
          <div>
            <span>订阅源</span>
            <strong>{subscriptionSummary.activeSources} / {subscriptionSummary.totalSources}</strong>
          </div>
          <div>
            <span>外部节点</span>
            <strong>{subscriptionSummary.enabledNodes} / {subscriptionSummary.totalNodes}</strong>
          </div>
          <div>
            <span>最近导入</span>
            <strong>{formatTime(subscriptionSummary.lastFetchedAt)}</strong>
          </div>
          <div>
            <span>异常源</span>
            <strong>{subscriptionSummary.errorSources}</strong>
          </div>
        </div>
      </section>

      <div className="workspace-grid two">
        <section className="panel">
          <SectionHeader eyebrow="Webhook" title="告警投递" />
          <form className="control-form" onSubmit={handleCreateWebhook}>
            <label>
              名称
              <input value={webhook.name} onChange={(event) => setWebhook({ ...webhook, name: event.target.value })} />
            </label>
            <label>
              类型
              <select value={webhook.kind} onChange={(event) => setWebhook({ ...webhook, kind: event.target.value })}>
                <option value="generic">Generic</option>
                <option value="telegram">Telegram</option>
                <option value="serverchan">Server 酱</option>
              </select>
            </label>
            <label>
              URL
              <input value={webhook.url} onChange={(event) => setWebhook({ ...webhook, url: event.target.value })} />
            </label>
            <label>
              Secret / Bot Token
              <input value={webhook.secret} onChange={(event) => setWebhook({ ...webhook, secret: event.target.value })} />
            </label>
            <label>
              Chat ID
              <input value={webhook.chatId} onChange={(event) => setWebhook({ ...webhook, chatId: event.target.value })} />
            </label>
            <label>
              事件类型
              <input value={webhook.eventTypes} onChange={(event) => setWebhook({ ...webhook, eventTypes: event.target.value })} />
            </label>
            <button className="primary-button" disabled={Boolean(busy) || controlsDisabled} type="submit">
              保存通道
            </button>
          </form>
          <WebhookList
            busy={Boolean(busy) || controlsDisabled}
            onTest={(id) => void runAction("Webhook 测试", () => testWebhook(id))}
            webhooks={control?.webhooks ?? []}
          />
        </section>

        <section className="panel">
          <SectionHeader eyebrow="订阅源" title="外部订阅聚合" />
          <form className="control-form" onSubmit={handleCreateSubscription}>
            <label>
              名称
              <input value={subscription.name} onChange={(event) => setSubscription({ ...subscription, name: event.target.value })} />
            </label>
            <label>
              URL
              <input value={subscription.url} onChange={(event) => setSubscription({ ...subscription, url: event.target.value })} />
            </label>
            <label className="full-span">
              内联内容
              <textarea value={subscription.content} onChange={(event) => setSubscription({ ...subscription, content: event.target.value })} />
            </label>
            <button className="primary-button" disabled={Boolean(busy) || controlsDisabled} type="submit">
              添加订阅源
            </button>
          </form>
          <MiniTable
            columns={["订阅源", "格式", "状态", "最近导入"]}
            rows={(control?.subscriptions ?? []).slice(0, 6).map((item) => [
              item.name,
              item.format || "auto",
              item.lastError || (item.enabled ? "启用" : "暂停"),
              formatTime(item.lastFetchedAt)
            ])}
          />
        </section>
      </div>

      <section className="panel">
        <SectionHeader eyebrow="聚合输出" title="订阅托管地址" />
        <div className="aggregate-box">
          <div className="button-row">
            <select
              aria-label="聚合订阅格式"
              value={aggregateFormat}
              onChange={(event) => setAggregateFormat(event.target.value as AggregateSubscriptionFormat)}
            >
              <option value="clash">Clash YAML</option>
              <option value="v2ray">V2Ray Base64</option>
              <option value="raw">Raw Shares</option>
              <option value="sing-box">Sing-box JSON</option>
            </select>
            <button className="ghost-button" disabled={Boolean(busy) || controlsDisabled} onClick={handleLoadAggregateSubscription} type="button">
              生成聚合订阅
            </button>
          </div>
          <code>{aggregateSubscriptionURL(aggregateFormat)}</code>
          {aggregateContent ? <textarea readOnly value={aggregateContent} /> : null}
        </div>
      </section>

      <section className="panel">
        <SectionHeader eyebrow="订阅节点" title="外部节点池" />
        <MiniTable
          columns={["节点", "协议", "地址", "质量"]}
          rows={(control?.externalNodes ?? []).slice(0, 8).map((node) => [
            node.name,
            node.protocol,
            `${node.address}:${node.port}`,
            `${node.latencyMs ?? "--"} ms / ${node.lossPercent ?? 0}%`
          ])}
        />
      </section>

      <AlertList alerts={control?.alerts ?? []} />
    </div>
  );
}

type SubscriptionFormatPreview = {
  description: string;
  label: string;
  value: AggregateSubscriptionFormat;
};

const subscriptionFormats: SubscriptionFormatPreview[] = [
  { value: "clash", label: "Clash YAML", description: "Rule Provider 与 Proxy Group 直接可用" },
  { value: "v2ray", label: "V2Ray Base64", description: "兼容通用订阅客户端" },
  { value: "raw", label: "Raw Shares", description: "调试和迁移时快速检查节点" },
  { value: "sing-box", label: "Sing-box JSON", description: "预留 sing-box 客户端托管输出" }
];

function buildSubscriptionSummary(control: DashboardDTO["control"] | undefined) {
  const subscriptions = control?.subscriptions ?? [];
  const nodes = control?.externalNodes ?? [];
  const lastFetched = subscriptions
    .map((item) => item.lastFetchedAt)
    .filter((value): value is string => Boolean(value))
    .sort((a, b) => new Date(b).getTime() - new Date(a).getTime())[0];

  return {
    activeSources: subscriptions.filter((item) => item.enabled).length,
    enabledNodes: nodes.filter((item) => item.enabled).length,
    errorSources: subscriptions.filter((item) => item.lastError).length,
    lastFetchedAt: lastFetched,
    totalNodes: nodes.length,
    totalSources: subscriptions.length
  };
}

function formatAvailableNodes(count: number, language: "zh-CN" | "en"): string {
  return language === "zh-CN" ? `${count} 个可用节点` : `${count} available nodes`;
}

function formatSourceCount(count: number, language: "zh-CN" | "en"): string {
  return language === "zh-CN" ? `${count} 个订阅源` : `${count} sources`;
}

function AlertList({ alerts }: { alerts: AlertEvent[] }) {
  const formatTime = useFormatTime();
  return (
    <section className="panel">
      <SectionHeader eyebrow="告警事件" title="最近投递状态" />
      <MiniTable
        columns={["事件", "来源", "级别", "状态", "时间"]}
        rows={alerts.slice(0, 8).map((alert) => [
          alert.eventType,
          `${alert.sourceType}:${alert.sourceId}`,
          alert.severity,
          alert.delivered ? "已投递" : alert.lastError || "待处理",
          formatTime(alert.createdAt)
        ])}
      />
    </section>
  );
}

function WebhookList({
  webhooks,
  busy,
  onTest
}: {
  busy: boolean;
  onTest: (id: string) => void;
  webhooks: WebhookEndpoint[];
}) {
  return (
    <div className="webhook-list">
      {webhooks.slice(0, 5).map((hook) => (
        <article key={hook.id}>
          <div>
            <strong>{hook.name}</strong>
            <span>
              {hook.kind} / {hook.enabled ? "启用" : "暂停"}
            </span>
          </div>
          <button className="ghost-button" disabled={busy || !hook.enabled} onClick={() => onTest(hook.id)} type="button">
            测试
          </button>
        </article>
      ))}
      {webhooks.length === 0 ? <p className="empty-state">暂无 Webhook</p> : null}
    </div>
  );
}

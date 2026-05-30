import { useState, type FormEvent } from "react";
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
  formatTime,
  MiniTable,
  NoticeRow,
  parseCSV,
  SectionHeader,
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
            columns={["节点", "协议", "地址"]}
            rows={(control?.externalNodes ?? []).slice(0, 6).map((node) => [
              node.name,
              node.protocol,
              `${node.address}:${node.port}`
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

      <AlertList alerts={control?.alerts ?? []} />
    </div>
  );
}

function AlertList({ alerts }: { alerts: AlertEvent[] }) {
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

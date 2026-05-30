import { useEffect, useState, type FormEvent } from "react";
import {
  askCopilot,
  createAPIKey,
  listAPIKeys,
  revokeAPIKey,
  updateAPIKey,
  type APIKey,
  type ControlTask,
  type DashboardDTO
} from "../api";
import {
  isAPIKeyResponse,
  MiniTable,
  NoticeRow,
  SectionHeader,
  taskTone,
  useFormatTime,
  ViewHeading
} from "./ConsolePrimitives";

type IntegrationsWorkspaceProps = {
  data: DashboardDTO | null;
  disabled?: boolean;
  onRefresh?: () => void;
};

const apiKeyScopeOptions = [
  { value: "panel:read", label: "只读 API", meta: "查询租户、节点、流量和订阅" },
  { value: "panel:write", label: "读写 API", meta: "允许创建任务、写入配置和集成回调" },
  { value: "panel:*", label: "控制面通配", meta: "保留给可信计费或运维系统" }
];

export function IntegrationsWorkspace({ data, disabled = false, onRefresh }: IntegrationsWorkspaceProps) {
  const formatTime = useFormatTime();
  const [busy, setBusy] = useState("");
  const [message, setMessage] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [apiKeysLoading, setAPIKeysLoading] = useState(false);
  const [apiKeysError, setAPIKeysError] = useState("");
  const [keyForm, setKeyForm] = useState({
    name: "计费系统集成",
    tenantId: "",
    scopes: ["panel:read"],
    expiresAt: ""
  });
  const [question, setQuestion] = useState("为什么最新 Agent 处于 degraded 状态？");
  const controlsDisabled = disabled || !data;

  useEffect(() => {
    if (!data) {
      return;
    }
    void refreshAPIKeys();
  }, [data]);

  async function refreshAPIKeys() {
    setAPIKeysLoading(true);
    setAPIKeysError("");
    try {
      const result = await listAPIKeys();
      setApiKeys(result.items);
    } catch (err) {
      setApiKeys([]);
      setAPIKeysError(err instanceof Error ? err.message : "API Key 列表加载失败");
    } finally {
      setAPIKeysLoading(false);
    }
  }

  async function runAction(label: string, action: () => Promise<unknown>, refreshKeys = false) {
    setBusy(label);
    setMessage("");
    setApiKey("");
    try {
      const result = await action();
      if (isAPIKeyResponse(result)) {
        setApiKey(result.apiKey);
      }
      if (refreshKeys) {
        await refreshAPIKeys();
      }
      setMessage(`${label} 已完成`);
      await Promise.resolve(onRefresh?.());
    } catch (err) {
      setMessage(err instanceof Error ? err.message : `${label} 执行失败`);
    } finally {
      setBusy("");
    }
  }

  function handleCreateAPIKey(event: FormEvent) {
    event.preventDefault();
    void runAction(
      "API Key",
      () =>
        createAPIKey({
          name: keyForm.name,
          tenantId: keyForm.tenantId,
          scopes: keyForm.scopes,
          status: "active",
          expiresAt: keyForm.expiresAt ? new Date(`${keyForm.expiresAt}T00:00:00Z`).toISOString() : undefined
        }),
      true
    );
  }

  function handleAskCopilot(event: FormEvent) {
    event.preventDefault();
    void runAction("AI Copilot", () => askCopilot(question));
  }

  function setKeyStatus(key: APIKey, status: "active" | "paused") {
    void runAction(status === "active" ? "启用密钥" : "暂停密钥", () => updateAPIKey(key.id, { status }), true);
  }

  function handleRevokeKey(key: APIKey) {
    void runAction("吊销密钥", () => revokeAPIKey(key.id), true);
  }

  return (
    <div className="workspace-view">
      <ViewHeading
        description="为第三方计费、发卡和运维系统预留 REST API 接入，并通过 AI Copilot 汇总异常流量与错误日志。"
        eyebrow="开放集成"
        title="API Key 与 AI 运维"
      />
      {message ? (
        <NoticeRow>
          <strong>{message}</strong>
          {apiKey ? <code>{apiKey}</code> : null}
        </NoticeRow>
      ) : null}

      <div className="workspace-grid two">
        <section className="panel">
          <SectionHeader eyebrow="REST API" title="签发作用域密钥" />
          <form className="control-form" onSubmit={handleCreateAPIKey}>
            <label>
              名称
              <input value={keyForm.name} onChange={(event) => setKeyForm({ ...keyForm, name: event.target.value })} />
            </label>
            <label>
              租户
              <select value={keyForm.tenantId} onChange={(event) => setKeyForm({ ...keyForm, tenantId: event.target.value })}>
                <option value="">主租户 / 全局</option>
                {(data?.control.tenants ?? []).map((tenant) => (
                  <option key={tenant.id} value={tenant.id}>
                    {tenant.name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              过期日期
              <input type="date" value={keyForm.expiresAt} onChange={(event) => setKeyForm({ ...keyForm, expiresAt: event.target.value })} />
            </label>
            <ScopePicker
              disabled={Boolean(busy) || controlsDisabled}
              onChange={(scopes) => setKeyForm({ ...keyForm, scopes })}
              value={keyForm.scopes}
            />
            <button className="primary-button" disabled={Boolean(busy) || controlsDisabled || keyForm.scopes.length === 0} type="submit">
              签发密钥
            </button>
          </form>
          <TaskRail tasks={data?.control.tasks ?? []} />
        </section>

        <section className="panel">
          <SectionHeader eyebrow="Copilot" title="故障排查问答" />
          <form className="control-form" onSubmit={handleAskCopilot}>
            <label className="full-span">
              问题
              <textarea value={question} onChange={(event) => setQuestion(event.target.value)} />
            </label>
            <button className="primary-button" disabled={Boolean(busy) || controlsDisabled} type="submit">
              询问 Copilot
            </button>
          </form>
          <div className="incident-list">
            {(data?.control.copilotIncidents ?? []).slice(0, 4).map((incident) => (
              <article key={incident.id}>
                <strong>{incident.question}</strong>
                <span>
                  {incident.model} / {incident.status} / {formatTime(incident.createdAt)}
                </span>
                <p>{incident.answer}</p>
              </article>
            ))}
          </div>
        </section>
      </div>

      <section className="panel">
        <SectionHeader eyebrow="API Governance" title="密钥治理台" />
        <APIKeyDesk
          error={apiKeysError}
          formatTime={formatTime}
          keys={apiKeys}
          loading={apiKeysLoading}
          onPause={(key) => setKeyStatus(key, "paused")}
          onResume={(key) => setKeyStatus(key, "active")}
          onRevoke={handleRevokeKey}
          tenants={data?.control.tenants ?? []}
        />
      </section>

      <section className="panel">
        <SectionHeader eyebrow="OpenAPI" title="第三方对接预留" />
        <MiniTable
          columns={["资源", "用途", "认证"]}
          rows={[
            ["/api/v1/api-docs", "OpenAPI 文档", "Bearer Token / API Key"],
            ["/api/v1/api-keys", "签发与治理集成密钥", "owner"],
            ["/api/v1/api-keys/{id}", "更新或吊销密钥", "owner"],
            ["/api/v1/copilot/ask", "AI 运维问答", "panel:write"],
            ["/api/v1/subscriptions/aggregate", "聚合订阅", "panel:read"]
          ]}
        />
      </section>
    </div>
  );
}

function ScopePicker({
  disabled,
  onChange,
  value
}: {
  disabled?: boolean;
  onChange: (scopes: string[]) => void;
  value: string[];
}) {
  function toggle(scope: string) {
    if (disabled) {
      return;
    }
    const selected = value.includes(scope);
    const next = selected ? value.filter((item) => item !== scope) : [...value, scope];
    onChange(next.length ? next : ["panel:read"]);
  }

  return (
    <fieldset className="scope-picker full-span" disabled={disabled}>
      <legend>作用域</legend>
      <div>
        {apiKeyScopeOptions.map((scope) => (
          <button className={value.includes(scope.value) ? "selected" : ""} key={scope.value} onClick={() => toggle(scope.value)} type="button">
            <strong>{scope.label}</strong>
            <span>{scope.value}</span>
            <em>{scope.meta}</em>
          </button>
        ))}
      </div>
    </fieldset>
  );
}

function APIKeyDesk({
  error,
  formatTime,
  keys,
  loading,
  onPause,
  onResume,
  onRevoke,
  tenants
}: {
  error: string;
  formatTime: (value?: string) => string;
  keys: APIKey[];
  loading: boolean;
  onPause: (key: APIKey) => void;
  onResume: (key: APIKey) => void;
  onRevoke: (key: APIKey) => void;
  tenants: DashboardDTO["control"]["tenants"];
}) {
  if (loading) {
    return (
      <div className="api-key-grid">
        {[0, 1, 2].map((item) => (
          <div className="api-key-card api-key-skeleton" key={item}>
            <span />
            <strong />
            <em />
          </div>
        ))}
      </div>
    );
  }
  if (error) {
    return <NoticeRow tone="danger">{error}</NoticeRow>;
  }
  if (keys.length === 0) {
    return <p className="empty-state">暂无 API Key，签发后会在这里进行暂停、启用和吊销。</p>;
  }
  return (
    <div className="api-key-grid">
      {keys.map((key) => (
        <article className="api-key-card" key={key.id}>
          <div className="api-key-card-head">
            <div>
              <strong>{key.name}</strong>
              <span>{key.id}</span>
            </div>
            <span className={`tenant-risk tenant-risk-${key.status === "active" ? "ok" : key.status === "revoked" ? "danger" : "warning"}`}>
              {key.status}
            </span>
          </div>
          <div className="api-key-meta">
            <span>租户</span>
            <strong>{tenantName(key.tenantId, tenants)}</strong>
          </div>
          <div className="api-key-meta">
            <span>最后使用</span>
            <strong>{formatTime(key.lastUsedAt)}</strong>
          </div>
          <div className="api-key-meta">
            <span>过期</span>
            <strong>{key.expiresAt ? formatTime(key.expiresAt) : "长期有效"}</strong>
          </div>
          <div className="tenant-scope-chips">
            {(key.scopes ?? []).map((scope) => (
              <span key={scope}>{scope}</span>
            ))}
          </div>
          <div className="api-key-actions">
            {key.status === "active" ? (
              <button className="ghost-button" onClick={() => onPause(key)} type="button">
                暂停
              </button>
            ) : key.status !== "revoked" ? (
              <button className="ghost-button" onClick={() => onResume(key)} type="button">
                启用
              </button>
            ) : null}
            <button className="ghost-button danger" disabled={key.status === "revoked"} onClick={() => onRevoke(key)} type="button">
              吊销
            </button>
          </div>
        </article>
      ))}
    </div>
  );
}

function tenantName(tenantId: string | undefined, tenants: DashboardDTO["control"]["tenants"]) {
  if (!tenantId) {
    return "主租户 / 全局";
  }
  return tenants.find((tenant) => tenant.id === tenantId)?.name ?? tenantId;
}

function TaskRail({ tasks }: { tasks: ControlTask[] }) {
  return (
    <div className="task-rail">
      {tasks.slice(0, 8).map((task) => (
        <span className={`task-state task-state-${taskTone(task.status)}`} key={task.id} title={task.lastError || task.type}>
          {task.type} / {task.status}
        </span>
      ))}
      {tasks.length === 0 ? <p className="empty-state">暂无任务</p> : null}
    </div>
  );
}

import { useState, type FormEvent } from "react";
import { askCopilot, createAPIKey, type ControlTask, type DashboardDTO } from "../api";
import {
  isAPIKeyResponse,
  MiniTable,
  NoticeRow,
  parseCSV,
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

export function IntegrationsWorkspace({ data, disabled = false, onRefresh }: IntegrationsWorkspaceProps) {
  const formatTime = useFormatTime();
  const [busy, setBusy] = useState("");
  const [message, setMessage] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [keyForm, setKeyForm] = useState({
    name: "计费系统集成",
    tenantId: "",
    scopes: "panel:read"
  });
  const [question, setQuestion] = useState("为什么最新 Agent 处于 degraded 状态？");
  const controlsDisabled = disabled || !data;

  async function runAction(label: string, action: () => Promise<unknown>) {
    setBusy(label);
    setMessage("");
    setApiKey("");
    try {
      const result = await action();
      if (isAPIKeyResponse(result)) {
        setApiKey(result.apiKey);
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
    void runAction("API Key", () =>
      createAPIKey({
        name: keyForm.name,
        tenantId: keyForm.tenantId,
        scopes: parseCSV(keyForm.scopes),
        status: "active"
      })
    );
  }

  function handleAskCopilot(event: FormEvent) {
    event.preventDefault();
    void runAction("AI Copilot", () => askCopilot(question));
  }

  return (
    <div className="workspace-view">
      <ViewHeading
        description="为第三方系统预留 REST API 接入，并通过 AI Copilot 汇总异常流量与错误日志。"
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
              租户 ID
              <input value={keyForm.tenantId} onChange={(event) => setKeyForm({ ...keyForm, tenantId: event.target.value })} />
            </label>
            <label>
              作用域
              <input value={keyForm.scopes} onChange={(event) => setKeyForm({ ...keyForm, scopes: event.target.value })} />
            </label>
            <button className="primary-button" disabled={Boolean(busy) || controlsDisabled} type="submit">
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
        <SectionHeader eyebrow="OpenAPI" title="第三方对接预留" />
        <MiniTable
          columns={["资源", "用途", "认证"]}
          rows={[
            ["/api/v1/api-docs", "OpenAPI 文档", "Bearer Token / API Key"],
            ["/api/v1/api-keys", "签发集成密钥", "panel:write"],
            ["/api/v1/copilot/ask", "AI 运维问答", "panel:write"],
            ["/api/v1/subscriptions/aggregate", "聚合订阅", "panel:read"]
          ]}
        />
      </section>
    </div>
  );
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

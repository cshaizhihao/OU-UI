import { useState, type FormEvent } from "react";
import {
  applyRouting,
  createRoutingRule,
  optimizeAgent,
  type DashboardDTO
} from "../api";
import type { Agent } from "../data";
import {
  formatServiceStatus,
  MiniTable,
  NoticeRow,
  SectionHeader,
  ViewHeading
} from "./ConsolePrimitives";

type RoutingWorkspaceProps = {
  agents: Agent[];
  data: DashboardDTO | null;
  disabled?: boolean;
  onRefresh?: () => void;
};

export function RoutingWorkspace({ agents, data, disabled = false, onRefresh }: RoutingWorkspaceProps) {
  const [busy, setBusy] = useState("");
  const [message, setMessage] = useState("");
  const [routing, setRouting] = useState({
    name: "阻断广告",
    ruleType: "ads",
    match: "category-ads-all",
    action: "block",
    priority: 10
  });
  const controlsDisabled = disabled || !data;

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

  function handleCreateRouting(event: FormEvent) {
    event.preventDefault();
    void runAction("路由规则", () =>
      createRoutingRule({
        ...routing,
        enabled: true,
        protocol: "",
        targetTag: routing.action === "proxy" ? "OU-Auto" : "",
        description: "由 OU-UI 控制台创建"
      })
    );
  }

  function handleApplyRouting() {
    const agentIds = agents.filter((agent) => agent.status !== "offline").map((agent) => agent.id);
    void runAction("下发分流", () => applyRouting(agentIds));
  }

  function handleOptimize() {
    const agent = agents.find((item) => item.status !== "offline") ?? agents[0];
    if (!agent) {
      setMessage("暂无可执行主机调优的 Agent");
      return;
    }
    void runAction("BBR v3 调优", () => optimizeAgent(agent.id));
  }

  return (
    <div className="workspace-view">
      <ViewHeading
        actions={
          <>
            <button className="ghost-button" disabled={Boolean(busy) || controlsDisabled} onClick={handleOptimize} type="button">
              一键 BBR v3
            </button>
            <button className="primary-button" disabled={Boolean(busy) || controlsDisabled} onClick={handleApplyRouting} type="button">
              下发到在线 Agent
            </button>
          </>
        }
        description="用可视化规则直接生成 Xray 路由 payload，并通过任务队列下发到具备能力的 Agent。"
        eyebrow="路由分流"
        title="GeoIP / GeoSite / 协议规则"
      />

      {message ? <NoticeRow>{message}</NoticeRow> : null}

      <div className="workspace-grid two">
        <section className="panel">
          <SectionHeader eyebrow="规则编辑器" title="新增分流规则" />
          <form className="control-form" onSubmit={handleCreateRouting}>
            <label>
              名称
              <input value={routing.name} onChange={(event) => setRouting({ ...routing, name: event.target.value })} />
            </label>
            <label>
              类型
              <select value={routing.ruleType} onChange={(event) => setRouting({ ...routing, ruleType: event.target.value })}>
                <option value="geoip">GeoIP</option>
                <option value="geosite">GeoSite</option>
                <option value="ads">广告过滤</option>
                <option value="domain">域名</option>
                <option value="protocol">协议</option>
                <option value="ip">IP CIDR</option>
              </select>
            </label>
            <label>
              匹配内容
              <input value={routing.match} onChange={(event) => setRouting({ ...routing, match: event.target.value })} />
            </label>
            <label>
              动作
              <select value={routing.action} onChange={(event) => setRouting({ ...routing, action: event.target.value })}>
                <option value="block">阻断</option>
                <option value="direct">直连</option>
                <option value="proxy">代理</option>
              </select>
            </label>
            <button className="primary-button" disabled={Boolean(busy) || controlsDisabled} type="submit">
              保存规则
            </button>
          </form>
        </section>

        <section className="panel">
          <SectionHeader eyebrow="当前规则" title="已配置分流" />
          <MiniTable
            columns={["规则", "匹配", "动作", "状态"]}
            rows={(data?.control.routingRules ?? []).slice(0, 8).map((rule) => [
              rule.name,
              `${rule.ruleType}:${rule.match}`,
              rule.action,
              rule.enabled ? "启用" : "停用"
            ])}
          />
        </section>
      </div>

      <section className="panel">
        <SectionHeader eyebrow="下发目标" title="在线 Agent 能力" />
        <MiniTable
          columns={["Agent", "状态", "区域", "能力"]}
          rows={agents.map((agent) => [
            agent.name,
            formatServiceStatus(agent.status),
            agent.region,
            agent.capabilities?.slice(0, 3).join(", ") || "未上报"
          ])}
        />
      </section>
    </div>
  );
}

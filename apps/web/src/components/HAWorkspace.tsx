import { useState, type FormEvent } from "react";
import { createLoadBalancer, type DashboardDTO } from "../api";
import type { Agent } from "../data";
import { MiniTable, NoticeRow, SectionHeader, ViewHeading } from "./ConsolePrimitives";

type HAWorkspaceProps = {
  agents: Agent[];
  data: DashboardDTO | null;
  disabled?: boolean;
  onRefresh?: () => void;
};

export function HAWorkspace({ agents, data, disabled = false, onRefresh }: HAWorkspaceProps) {
  const [busy, setBusy] = useState("");
  const [message, setMessage] = useState("");
  const [loadBalancer, setLoadBalancer] = useState({
    name: "全球入口 HA",
    entryTag: "ou-ha",
    strategy: "latency-loss",
    healthCheckInterval: 30
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

  function handleCreateLoadBalancer(event: FormEvent) {
    event.preventDefault();
    const members = agents.slice(0, 4).map((agent, index) => ({
      id: agent.id,
      agentId: agent.id,
      name: agent.name,
      address: agent.ip,
      port: 443,
      latencyMs: 40 + index * 15,
      lossPercent: index === 0 ? 0 : 0.2,
      weight: agent.status === "online" ? 2 : 1,
      status: agent.status === "offline" ? "offline" : "healthy"
    }));
    void runAction("高可用组", () =>
      createLoadBalancer({
        ...loadBalancer,
        members,
        healthCheckInterval: Number(loadBalancer.healthCheckInterval) || 30
      })
    );
  }

  return (
    <div className="workspace-view">
      <ViewHeading
        description="将多个 Agent 聚合为统一入口，根据延迟、丢包和权重动态选择后端。"
        eyebrow="高可用"
        title="负载均衡组"
      />
      {message ? <NoticeRow>{message}</NoticeRow> : null}

      <div className="workspace-grid two">
        <section className="panel">
          <SectionHeader eyebrow="均衡组" title="创建入口组" />
          <form className="control-form" onSubmit={handleCreateLoadBalancer}>
            <label>
              组名
              <input value={loadBalancer.name} onChange={(event) => setLoadBalancer({ ...loadBalancer, name: event.target.value })} />
            </label>
            <label>
              入口标识
              <input value={loadBalancer.entryTag} onChange={(event) => setLoadBalancer({ ...loadBalancer, entryTag: event.target.value })} />
            </label>
            <label>
              策略
              <select value={loadBalancer.strategy} onChange={(event) => setLoadBalancer({ ...loadBalancer, strategy: event.target.value })}>
                <option value="latency-loss">延迟 + 丢包</option>
                <option value="weighted">权重优先</option>
              </select>
            </label>
            <label>
              探测间隔秒
              <input
                type="number"
                value={loadBalancer.healthCheckInterval}
                onChange={(event) => setLoadBalancer({ ...loadBalancer, healthCheckInterval: Number(event.target.value) })}
              />
            </label>
            <button className="primary-button" disabled={Boolean(busy) || controlsDisabled} type="submit">
              创建 HA 组
            </button>
          </form>
        </section>

        <section className="panel">
          <SectionHeader eyebrow="决策" title="当前入口选择" />
          <MiniTable
            columns={["组", "入口", "选中后端", "得分"]}
            rows={(data?.control.loadBalancers ?? []).slice(0, 8).map((group) => [
              group.name,
              group.entryTag,
              String(group.lastDecision?.selected ?? "-"),
              Number(group.lastDecision?.score ?? 0).toFixed(1)
            ])}
          />
        </section>
      </div>
    </div>
  );
}

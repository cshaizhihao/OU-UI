import type { DashboardDTO } from "../api";
import type { Agent } from "../data";
import { protocolOptions, runtimeOptions } from "../data";
import { AgentTable, ProbeAgentCards } from "./AgentViews";
import { formatServiceStatus, KpiGrid, MiniTable, SectionHeader, ViewHeading } from "./ConsolePrimitives";

type NodeManagementWorkspaceProps = {
  agents: Agent[];
  data: DashboardDTO | null;
};

export function NodeManagementWorkspace({ agents, data }: NodeManagementWorkspaceProps) {
  const nodes = data?.control.nodes ?? [];
  const onlineAgents = agents.filter((agent) => agent.status === "online").length;
  const activeNodes = nodes.filter((node) => !["failed", "offline", "disabled"].includes(node.status)).length;
  const queuedTasks = data?.control.tasks.filter((task) => ["pending", "queued", "running"].includes(task.status)).length ?? 0;

  return (
    <div className="workspace-view">
      <ViewHeading
        description="这里集中处理 Agent、生成节点、运行时部署和服务状态，不再和其他业务上下堆叠。"
        eyebrow="节点管理"
        title="Agent 与托管节点"
      />

      <KpiGrid
        items={[
          { label: "在线 Agent", value: `${onlineAgents} / ${agents.length}`, delta: "心跳在线" },
          { label: "健康节点", value: `${activeNodes} / ${nodes.length}`, delta: "运行时状态" },
          { label: "任务队列", value: String(queuedTasks), delta: "部署与调优任务" },
          { label: "协议能力", value: String(protocolOptions.length), delta: runtimeOptions.join(" / ") }
        ]}
      />

      <section className="panel">
        <SectionHeader eyebrow="节点下发" title="运行时服务控制" />
        <form className="dispatch-form">
          <label>
            Agent
            <select defaultValue={agents[0]?.id ?? ""} disabled={!agents.length}>
              {agents.length ? (
                agents.map((agent) => (
                  <option value={agent.id} key={agent.id}>
                    {agent.name} - {agent.region}
                  </option>
                ))
              ) : (
                <option value="">暂无可用 Agent</option>
              )}
            </select>
          </label>
          <label>
            运行时
            <select defaultValue="Xray">
              {runtimeOptions.map((runtime) => (
                <option value={runtime} key={runtime}>
                  {runtime}
                </option>
              ))}
            </select>
          </label>
          <label>
            协议
            <select defaultValue="VLESS Reality">
              {protocolOptions.map((protocol) => (
                <option value={protocol} key={protocol}>
                  {protocol}
                </option>
              ))}
            </select>
          </label>
          <label>
            队列策略
            <select defaultValue="rolling">
              <option value="rolling">托管 reload，保持活动连接</option>
              <option value="immediate">托管 restart，维护窗口执行</option>
              <option value="staged">外部服务，等待人工确认</option>
            </select>
          </label>
        </form>
      </section>

      <ProbeAgentCards agents={agents} />

      <section className="panel">
        <SectionHeader eyebrow="托管节点" title="节点健康" />
        <MiniTable
          columns={["节点", "Agent", "运行时", "协议", "状态"]}
          rows={nodes.slice(0, 10).map((node) => [
            node.name,
            node.agentId,
            node.runtime,
            node.protocol,
            formatServiceStatus(node.serviceStatus || node.status)
          ])}
        />
      </section>

      <AgentTable agents={agents} />
    </div>
  );
}

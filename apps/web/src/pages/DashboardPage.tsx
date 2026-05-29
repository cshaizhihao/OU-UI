import { AgentCards, AgentTable } from "../components/AgentViews";
import { AnalyticsPanel } from "../components/Charts";
import { agents, nodeRows, taskRows } from "../data";

const kpis = [
  { label: "今日任务", value: "1,284", delta: "+12.4%" },
  { label: "活跃 Agent", value: "36", delta: "+4" },
  { label: "平均延迟", value: "1.8s", delta: "-0.3s" },
  { label: "节点健康", value: "97.6%", delta: "+1.1%" }
];

export function DashboardPage() {
  return (
    <div className="dashboard">
      <section className="kpi-grid" id="概览">
        {kpis.map((kpi) => (
          <article className="kpi-card" key={kpi.label}>
            <span>{kpi.label}</span>
            <strong>{kpi.value}</strong>
            <em>{kpi.delta}</em>
          </article>
        ))}
      </section>

      <AnalyticsPanel />
      <AgentCards agents={agents} />

      <div className="split-grid">
        <section className="panel" id="任务">
          <div className="section-heading compact">
            <h2>任务页面占位</h2>
            <button className="ghost-button">查看全部</button>
          </div>
          <div className="task-list">
            {taskRows.map((task) => (
              <article className="task-item" key={task.name}>
                <div>
                  <strong>{task.name}</strong>
                  <span>{task.owner} · {task.state}</span>
                </div>
                <div className="progress">
                  <span style={{ width: `${task.progress}%` }} />
                </div>
              </article>
            ))}
          </div>
        </section>

        <section className="panel" id="节点">
          <div className="section-heading compact">
            <h2>节点页面占位</h2>
            <button className="ghost-button">扩容</button>
          </div>
          <div className="node-list">
            {nodeRows.map((node) => (
              <article className="node-item" key={node.name}>
                <div>
                  <strong>{node.name}</strong>
                  <span>{node.region}</span>
                </div>
                <meter min="0" max="100" value={node.load} />
                <small>{node.health}</small>
              </article>
            ))}
          </div>
        </section>
      </div>

      <AgentTable agents={agents} />
    </div>
  );
}

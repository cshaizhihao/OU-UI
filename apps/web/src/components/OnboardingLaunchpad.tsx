import { useMemo, useState } from "react";
import {
  agentInstallCommand,
  aggregateSubscriptionURL,
  createNode,
  loadAgentInstallScript,
  loadNodeShare,
  panelBaseURL,
  type AggregateSubscriptionFormat,
  type DashboardDTO,
  type NodeShare
} from "../api";
import type { Agent } from "../data";
import { launchpadCopy, starterStepDefinitions } from "../onboarding";
import { SectionHeader, StatusTag, useLocale } from "./ConsolePrimitives";

type OnboardingLaunchpadProps = {
  agents: Agent[];
  data: DashboardDTO | null;
  onRefresh?: () => void;
};

type StepTone = "ok" | "warning" | "muted";

export function OnboardingLaunchpad({ agents, data, onRefresh }: OnboardingLaunchpadProps) {
  const language = useLocale();
  const copy = launchpadCopy[language];
  const onlineAgents = agents.filter((agent) => agent.status === "online");
  const nodes = data?.control.nodes ?? [];
  const tasks = data?.control.tasks ?? [];
  const failedTasks = tasks.filter((task) => task.status === "failed").length;
  const latestTask = tasks[0];
  const [serverUrl, setServerUrl] = useState(() => panelBaseURL());
  const [selectedAgentId, setSelectedAgentId] = useState(() => onlineAgents[0]?.id ?? agents[0]?.id ?? "");
  const [nodeName, setNodeName] = useState("ou-ui-vless-01");
  const [nodePort, setNodePort] = useState(443);
  const [subscriptionFormat, setSubscriptionFormat] = useState<AggregateSubscriptionFormat>("clash");
  const [shareNodeId, setShareNodeId] = useState("");
  const [nodeShare, setNodeShare] = useState<NodeShare | null>(null);
  const [installScript, setInstallScript] = useState("");
  const [busy, setBusy] = useState("");
  const [notice, setNotice] = useState("");

  const selectedAgent = agents.find((agent) => agent.id === selectedAgentId) ?? onlineAgents[0] ?? agents[0];
  const selectedShareNode = nodes.find((node) => node.id === shareNodeId) ?? nodes[0];
  const command = useMemo(() => agentInstallCommand(serverUrl), [serverUrl]);
  const subscriptionUrl = useMemo(() => aggregateSubscriptionURL(subscriptionFormat), [subscriptionFormat]);
  const sharePreview = selectedShareNode && nodeShare?.nodeId === selectedShareNode.id ? nodeShare.share : subscriptionUrl;
  const aggregateAction = starterStepDefinitions[2].copy[language].action;
  const singleNodeAction = language === "zh" ? "复制节点链接" : "Copy node link";
  const nodeLinkLabel = language === "zh" ? "节点链接" : "Node link";
  const fallbackNotice = language === "zh" ? "节点链接暂不可用，已复制聚合订阅" : "Node link unavailable; copied aggregate subscription";
  const starterHealth = nodes.length > 0 && onlineAgents.length > 0 && failedTasks === 0;
  const stepTones = buildStepTones({
    failedTasks,
    hasAgent: onlineAgents.length > 0,
    hasNode: nodes.length > 0,
    scriptLoaded: Boolean(installScript)
  });

  async function copyValue(value: string, label: string = copy.copied) {
    try {
      await navigator.clipboard.writeText(value);
      setNotice(label);
    } catch {
      setNotice(copy.copyFailed);
    }
  }

  async function handleLoadScript() {
    setBusy("script");
    setNotice("");
    try {
      setInstallScript(await loadAgentInstallScript(serverUrl));
      setNotice(copy.loadScript);
    } catch (err) {
      setNotice(err instanceof Error ? err.message : copy.copyFailed);
    } finally {
      setBusy("");
    }
  }

  async function handleCreateNode() {
    if (!selectedAgent) {
      setNotice(copy.selectAgentFirst);
      return;
    }
    setBusy("node");
    setNotice("");
    try {
      await createNode({
        agentId: selectedAgent.id,
        name: nodeName || "ou-ui-vless",
        runtime: "xray",
        protocol: "vless",
        listen: "0.0.0.0",
        port: nodePort,
        settings: {
          encryption: "none",
          remark: nodeName || "ou-ui-vless",
          uuid: createClientUUID()
        }
      });
      setNotice(copy.nodeCreated);
      onRefresh?.();
    } catch (err) {
      setNotice(err instanceof Error ? err.message : copy.copyFailed);
    } finally {
      setBusy("");
    }
  }

  async function handleCopyShare() {
    if (!selectedShareNode) {
      await copyValue(subscriptionUrl, aggregateAction);
      return;
    }
    setBusy("share");
    setNotice("");
    try {
      const result = await loadNodeShare(selectedShareNode.id);
      setNodeShare(result);
      await copyValue(result.share, singleNodeAction);
    } catch {
      setNodeShare(null);
      await copyValue(subscriptionUrl, fallbackNotice);
    } finally {
      setBusy("");
    }
  }

  return (
    <div className="launchpad">
      <section className="launchpad-hero">
        <div>
          <p className="eyebrow">{copy.eyebrow}</p>
          <h2>{copy.title}</h2>
          <span>{copy.description}</span>
        </div>
        <StatusTag tone={starterHealth ? "ok" : "warning"}>
          {starterHealth ? copy.healthReady : copy.healthWaiting}
        </StatusTag>
      </section>

      <section className="starter-steps" aria-label={copy.title}>
        {starterStepDefinitions.map((step) => {
          const stepCopy = step.copy[language];
          return (
            <article className={`starter-step starter-step-${stepTones[step.id]}`} key={step.id}>
              <div className="starter-step-index">
                <span>{stepCopy.label}</span>
              </div>
              <div>
                <h3>{stepCopy.title}</h3>
                <p>{stepCopy.description}</p>
              </div>
            </article>
          );
        })}
      </section>

      <div className="launchpad-grid">
        <section className="panel launchpad-panel">
          <SectionHeader eyebrow="Step 01" title={starterStepDefinitions[0].copy[language].title} />
          <label>
            {copy.serverUrl}
            <input value={serverUrl} onChange={(event) => setServerUrl(event.target.value)} />
          </label>
          <div className="command-preview">
            <span>{copy.commandPreview}</span>
            <code>{command}</code>
          </div>
          <div className="button-row">
            <button className="primary-button" onClick={() => copyValue(command, starterStepDefinitions[0].copy[language].action)} type="button">
              {starterStepDefinitions[0].copy[language].action}
            </button>
            <button className="ghost-button" disabled={busy === "script"} onClick={handleLoadScript} type="button">
              {busy === "script" ? "..." : copy.loadScript}
            </button>
          </div>
          {installScript ? (
            <pre className="script-preview" aria-label={copy.scriptPreview}>
              {installScript}
            </pre>
          ) : null}
        </section>

        <section className="panel launchpad-panel">
          <SectionHeader eyebrow="Step 02" title={starterStepDefinitions[1].copy[language].title} />
          <label>
            {copy.agent}
            <select value={selectedAgent?.id ?? ""} onChange={(event) => setSelectedAgentId(event.target.value)} disabled={!agents.length}>
              {agents.length ? (
                agents.map((agent) => (
                  <option value={agent.id} key={agent.id}>
                    {agent.name} - {agent.status}
                  </option>
                ))
              ) : (
                <option value="">{copy.noAgent}</option>
              )}
            </select>
          </label>
          <div className="launchpad-form-row">
            <label>
              {copy.nodeName}
              <input value={nodeName} onChange={(event) => setNodeName(event.target.value)} />
            </label>
            <label>
              {copy.port}
              <input
                min={1}
                max={65535}
                type="number"
                value={nodePort}
                onChange={(event) => setNodePort(Number(event.target.value) || 443)}
              />
            </label>
          </div>
          <button className="primary-button" disabled={busy === "node" || !selectedAgent} onClick={handleCreateNode} type="button">
            {busy === "node" ? "..." : starterStepDefinitions[1].copy[language].action}
          </button>
        </section>

        <section className="panel launchpad-panel">
          <SectionHeader eyebrow="Step 03" title={starterStepDefinitions[2].copy[language].title} />
          {nodes.length ? (
            <label>
              {nodeLinkLabel}
              <select
                value={selectedShareNode?.id ?? ""}
                onChange={(event) => {
                  setShareNodeId(event.target.value);
                  setNodeShare(null);
                }}
              >
                {nodes.map((node) => (
                  <option value={node.id} key={node.id}>
                    {node.name} - {node.status}
                  </option>
                ))}
              </select>
            </label>
          ) : null}
          <label>
            {copy.subscriptionFormat}
            <select
              value={subscriptionFormat}
              onChange={(event) => setSubscriptionFormat(event.target.value as AggregateSubscriptionFormat)}
            >
              <option value="clash">Clash</option>
              <option value="v2ray">V2Ray</option>
              <option value="raw">Raw</option>
              <option value="sing-box">Sing-box</option>
            </select>
          </label>
          <div className="command-preview">
            <span>{selectedShareNode ? singleNodeAction : aggregateAction}</span>
            <code>{sharePreview}</code>
          </div>
          <div className="button-row">
            <button className="primary-button" disabled={busy === "share"} onClick={handleCopyShare} type="button">
              {busy === "share" ? "..." : selectedShareNode ? singleNodeAction : aggregateAction}
            </button>
            {selectedShareNode ? (
              <button className="ghost-button" onClick={() => copyValue(subscriptionUrl, aggregateAction)} type="button">
                {aggregateAction}
              </button>
            ) : null}
          </div>
        </section>

        <section className="panel launchpad-panel status-panel">
          <SectionHeader
            actions={
              <button className="ghost-button" onClick={onRefresh} type="button">
                {starterStepDefinitions[3].copy[language].action}
              </button>
            }
            eyebrow="Step 04"
            title={starterStepDefinitions[3].copy[language].title}
          />
          <div className="status-grid" aria-label={copy.statusSummary}>
            <StatusMetric label="Agent" value={`${onlineAgents.length} / ${agents.length}`} tone={onlineAgents.length ? "ok" : "warning"} />
            <StatusMetric label={language === "zh" ? "节点" : "Nodes"} value={String(nodes.length)} tone={nodes.length ? "ok" : "muted"} />
            <StatusMetric label={language === "zh" ? "失败任务" : "Failed tasks"} value={String(failedTasks)} tone={failedTasks ? "warning" : "ok"} />
            <StatusMetric label={language === "zh" ? "最近任务" : "Latest task"} value={latestTask?.status ?? "--"} tone={failedTasks ? "warning" : "muted"} />
          </div>
        </section>
      </div>

      {notice ? <div className="notice-row notice-success launchpad-notice">{notice}</div> : null}
    </div>
  );
}

function StatusMetric({ label, tone, value }: { label: string; tone: StepTone; value: string }) {
  return (
    <div className={`status-metric status-metric-${tone}`}>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function buildStepTones({
  failedTasks,
  hasAgent,
  hasNode,
  scriptLoaded
}: {
  failedTasks: number;
  hasAgent: boolean;
  hasNode: boolean;
  scriptLoaded: boolean;
}): Record<(typeof starterStepDefinitions)[number]["id"], StepTone> {
  return {
    "connect-agent": hasAgent || scriptLoaded ? "ok" : "warning",
    "create-node": hasNode ? "ok" : hasAgent ? "warning" : "muted",
    "copy-link": hasNode ? "ok" : "warning",
    "verify-status": hasAgent && hasNode && failedTasks === 0 ? "ok" : failedTasks ? "warning" : "muted"
  };
}

function createClientUUID(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (token) => {
    const value = Math.floor(Math.random() * 16);
    const next = token === "x" ? value : (value & 0x3) | 0x8;
    return next.toString(16);
  });
}

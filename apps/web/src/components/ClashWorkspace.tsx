import { useState, type FormEvent } from "react";
import { clashProfileURL, createClashProfile, type ClashProfile, type DashboardDTO } from "../api";
import {
  MiniTable,
  NoticeRow,
  parseCSV,
  parseLines,
  SectionHeader,
  stringsTrim,
  useFormatTime,
  useLocale,
  ViewHeading
} from "./ConsolePrimitives";

type ClashWorkspaceProps = {
  data: DashboardDTO | null;
  disabled?: boolean;
  onRefresh?: () => void;
};

export function ClashWorkspace({ data, disabled = false, onRefresh }: ClashWorkspaceProps) {
  const [busy, setBusy] = useState("");
  const [message, setMessage] = useState("");
  const [clash, setClash] = useState({
    name: "OU-UI 托管 Clash",
    providerName: "private",
    providerUrl: "https://example.com/rules/private.yaml",
    providerBehavior: "domain",
    groupName: "OU-Auto",
    groupType: "url-test",
    groupNodes: "*",
    selectedNodes: "*",
    routingRules: ""
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

  function handleCreateClash(event: FormEvent) {
    event.preventDefault();
    const providerUrl = stringsTrim(clash.providerUrl);
    const groupName = stringsTrim(clash.groupName);
    void runAction("Clash YAML", () =>
      createClashProfile({
        name: clash.name,
        ruleProviders: providerUrl
          ? [
              {
                name: stringsTrim(clash.providerName) || "private",
                type: "http",
                behavior: clash.providerBehavior,
                url: providerUrl,
                interval: 86400
              }
            ]
          : [],
        proxyGroups: groupName
          ? [
              {
                name: groupName,
                type: clash.groupType,
                proxies: parseCSV(clash.groupNodes),
                url: "https://www.gstatic.com/generate_204",
                interval: 300
              }
            ]
          : [],
        routingRules: parseLines(clash.routingRules),
        selectedNodes: parseCSV(clash.selectedNodes)
      })
    );
  }

  async function handleCopyProfile(path: string) {
    try {
      await navigator.clipboard.writeText(path);
      setMessage("托管地址已复制");
    } catch {
      setMessage(path);
    }
  }

  return (
    <div className="workspace-view">
      <ViewHeading
        description="独立管理 Rule Provider、Proxy Group、节点选择和完整 YAML 托管输出。"
        eyebrow="Clash 托管"
        title="规则生成工作区"
      />
      {message ? <NoticeRow>{message}</NoticeRow> : null}

      <div className="workspace-grid two">
        <section className="panel">
          <SectionHeader eyebrow="Rule Provider" title="规则源与分组" />
          <form className="control-form" onSubmit={handleCreateClash}>
            <label>
              配置名称
              <input value={clash.name} onChange={(event) => setClash({ ...clash, name: event.target.value })} />
            </label>
            <label>
              Provider 名称
              <input value={clash.providerName} onChange={(event) => setClash({ ...clash, providerName: event.target.value })} />
            </label>
            <label>
              Provider URL
              <input value={clash.providerUrl} onChange={(event) => setClash({ ...clash, providerUrl: event.target.value })} />
            </label>
            <label>
              Provider 类型
              <select value={clash.providerBehavior} onChange={(event) => setClash({ ...clash, providerBehavior: event.target.value })}>
                <option value="domain">Domain</option>
                <option value="ipcidr">IP CIDR</option>
                <option value="classical">Classical</option>
              </select>
            </label>
            <label>
              Proxy Group
              <input value={clash.groupName} onChange={(event) => setClash({ ...clash, groupName: event.target.value })} />
            </label>
            <label>
              分组类型
              <select value={clash.groupType} onChange={(event) => setClash({ ...clash, groupType: event.target.value })}>
                <option value="url-test">URL Test</option>
                <option value="fallback">Fallback</option>
                <option value="select">Select</option>
                <option value="load-balance">Load Balance</option>
              </select>
            </label>
            <label>
              分组节点
              <input value={clash.groupNodes} onChange={(event) => setClash({ ...clash, groupNodes: event.target.value })} />
            </label>
            <label>
              配置节点
              <input value={clash.selectedNodes} onChange={(event) => setClash({ ...clash, selectedNodes: event.target.value })} />
            </label>
            <label className="full-span">
              规则
              <textarea value={clash.routingRules} onChange={(event) => setClash({ ...clash, routingRules: event.target.value })} />
            </label>
            <button className="primary-button" disabled={Boolean(busy) || controlsDisabled} type="submit">
              生成 YAML
            </button>
          </form>
        </section>

        <section className="panel">
          <SectionHeader eyebrow="托管文件" title="已生成配置" />
          <ProfileList onCopy={(path) => void handleCopyProfile(path)} profiles={data?.control.clashProfiles ?? []} />
        </section>
      </div>

      <section className="panel">
        <SectionHeader eyebrow="候选节点" title="外部与自建节点" />
        <MiniTable
          columns={["节点", "协议", "地址"]}
          rows={(data?.control.externalNodes ?? []).slice(0, 8).map((node) => [
            node.name,
            node.protocol,
            `${node.address}:${node.port}`
          ])}
        />
      </section>
    </div>
  );
}

function ProfileList({ onCopy, profiles }: { onCopy: (path: string) => void; profiles: ClashProfile[] }) {
  const formatTime = useFormatTime();
  const language = useLocale();
  return (
    <div className="profile-list clash-profile-list">
      {profiles.slice(0, 8).map((profile) => (
        <article key={profile.id}>
          <div className="profile-card-head">
            <div>
              <strong>{profile.name}</strong>
              <span>{formatTime(profile.updatedAt)}</span>
            </div>
            <button className="ghost-button" onClick={() => onCopy(profilePath(profile.id))} type="button">
              复制地址
            </button>
          </div>
          <code>{clashProfileURL(profile.id)}</code>
          <div className="profile-card-stats">
            <span>Provider {profile.ruleProviders?.length ?? 0}</span>
            <span>Proxy Group {profile.proxyGroups?.length ?? 0}</span>
            <span>{language === "zh-CN" ? `规则 ${profile.routingRules?.length ?? 0}` : `Rules ${profile.routingRules?.length ?? 0}`}</span>
            <span>{formatYamlLineCount(profile.generatedYaml, language)}</span>
          </div>
          {profile.generatedYaml ? (
            <pre className="yaml-preview">{profile.generatedYaml.split("\n").slice(0, 8).join("\n")}</pre>
          ) : null}
        </article>
      ))}
      {profiles.length === 0 ? <p className="empty-state">暂无 Clash 配置</p> : null}
    </div>
  );
}

function profilePath(id: string): string {
  return clashProfileURL(id);
}

function formatYamlLineCount(value: string | undefined, language: "zh-CN" | "en"): string {
  if (!value) {
    return language === "zh-CN" ? "YAML 未生成" : "YAML not generated";
  }
  const lineCount = value.split("\n").length;
  return language === "zh-CN" ? `YAML ${lineCount} 行` : `YAML ${lineCount} lines`;
}

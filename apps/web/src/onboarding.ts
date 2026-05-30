import type { Locale, WorkspaceId } from "./components/Shell";

export const primaryWorkspaceId = "overview" as const;

export const starterStepIds = [
  "connect-agent",
  "create-node",
  "copy-link",
  "verify-status"
] as const;

export type StarterStepId = typeof starterStepIds[number];

export type StarterStepDefinition = {
  id: StarterStepId;
  copy: Record<
    Locale,
    {
      action: string;
      description: string;
      label: string;
      title: string;
    }
  >;
};

export const starterStepDefinitions: readonly StarterStepDefinition[] = [
  {
    id: "connect-agent",
    copy: {
      zh: {
        label: "01",
        title: "接入服务器",
        description: "复制一条安装命令，在服务器上运行后 Agent 会自动回连面板。",
        action: "复制安装命令"
      },
      en: {
        label: "01",
        title: "Connect server",
        description: "Copy one install command and run it on the server so the Agent checks in.",
        action: "Copy install command"
      }
    }
  },
  {
    id: "create-node",
    copy: {
      zh: {
        label: "02",
        title: "创建协议节点",
        description: "选择在线 Agent，生成一个可部署的 VLESS 节点并进入任务队列。",
        action: "创建节点"
      },
      en: {
        label: "02",
        title: "Create node",
        description: "Pick an online Agent and create a deployable VLESS node task.",
        action: "Create node"
      }
    }
  },
  {
    id: "copy-link",
    copy: {
      zh: {
        label: "03",
        title: "复制订阅或节点链接",
        description: "优先复制单节点链接，也可以继续复制 Clash、V2Ray 或 Sing-box 聚合订阅地址。",
        action: "复制订阅"
      },
      en: {
        label: "03",
        title: "Copy subscription or node link",
        description: "Copy a single node link first, with Clash, V2Ray, or Sing-box aggregate subscriptions still available.",
        action: "Copy subscription"
      }
    }
  },
  {
    id: "verify-status",
    copy: {
      zh: {
        label: "04",
        title: "查看状态",
        description: "确认 Agent 在线、节点健康、任务队列没有失败项。",
        action: "刷新状态"
      },
      en: {
        label: "04",
        title: "Check status",
        description: "Confirm Agents are online, nodes are healthy, and no task failed.",
        action: "Refresh status"
      }
    }
  }
] as const;

export const professionalWorkspaceIds = [
  "nodes",
  "traffic",
  "routing",
  "ha",
  "operations",
  "clash",
  "tenants",
  "integrations"
] as const satisfies readonly WorkspaceId[];

export const workspaceModeLabel = {
  zh: "专业模式",
  en: "Professional mode"
} as const satisfies Record<Locale, string>;

export const starterModeLabel = {
  zh: "默认启动台",
  en: "Default launchpad"
} as const satisfies Record<Locale, string>;

export const launchpadCopy = {
  zh: {
    eyebrow: "默认启动台",
    title: "四步完成 OU-UI 首次开通",
    description: "先把服务器接进来，再创建节点、复制订阅、确认状态。高级路由、HA、RBAC、API Key 与 Copilot 已收进左侧专业模式。",
    healthReady: "主流程正常",
    healthWaiting: "等待接入",
    copied: "已复制",
    copyFailed: "复制失败",
    loadScript: "查看脚本",
    serverUrl: "面板外网地址",
    agent: "目标 Agent",
    nodeName: "节点名称",
    port: "端口",
    noAgent: "暂无可用 Agent",
    selectAgentFirst: "请先接入或选择 Agent",
    nodeCreated: "节点创建任务已进入队列",
    subscriptionFormat: "订阅格式",
    commandPreview: "一键安装命令",
    scriptPreview: "安装脚本预览",
    statusSummary: "状态摘要"
  },
  en: {
    eyebrow: "Default launchpad",
    title: "Open OU-UI in four steps",
    description: "Connect a server, create a node, copy a subscription, then verify status. Routing, HA, RBAC, API keys, and Copilot live in Professional mode.",
    healthReady: "Primary flow ready",
    healthWaiting: "Waiting for intake",
    copied: "Copied",
    copyFailed: "Copy failed",
    loadScript: "View script",
    serverUrl: "Public panel URL",
    agent: "Target Agent",
    nodeName: "Node name",
    port: "Port",
    noAgent: "No Agent available",
    selectAgentFirst: "Connect or select an Agent first",
    nodeCreated: "Node creation task queued",
    subscriptionFormat: "Subscription format",
    commandPreview: "One-command install",
    scriptPreview: "Install script preview",
    statusSummary: "Status summary"
  }
} as const;

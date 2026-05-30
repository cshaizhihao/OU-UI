import type { ReactNode } from "react";
import type { SessionUser } from "../api";

export type Locale = "zh" | "en";
export type ThemeMode = "light" | "dark";
export type WorkspaceId =
  | "overview"
  | "nodes"
  | "traffic"
  | "routing"
  | "ha"
  | "operations"
  | "clash"
  | "tenants"
  | "integrations";

type WorkspaceItem = {
  id: WorkspaceId;
  copy: Record<
    Locale,
    {
      description: string;
      eyebrow: string;
      label: string;
      title: string;
    }
  >;
};

export const workspaceItems: WorkspaceItem[] = [
  {
    id: "overview",
    copy: {
      zh: {
        label: "总览工作台",
        eyebrow: "控制面总览",
        title: "OU-UI 运维控制台",
        description: "聚合 Agent、节点、告警和任务状态"
      },
      en: {
        label: "Overview",
        eyebrow: "Control Plane",
        title: "OU-UI Operations Console",
        description: "Unified view of Agents, nodes, alerts, and tasks"
      }
    }
  },
  {
    id: "nodes",
    copy: {
      zh: {
        label: "节点管理",
        eyebrow: "Agent 与节点",
        title: "节点管理",
        description: "管理 Agent、托管节点、运行时服务和任务队列"
      },
      en: {
        label: "Nodes",
        eyebrow: "Agents and Nodes",
        title: "Node Management",
        description: "Manage Agents, generated nodes, runtime services, and task queues"
      }
    }
  },
  {
    id: "traffic",
    copy: {
      zh: {
        label: "流量审计",
        eyebrow: "单节点采样",
        title: "流量审计",
        description: "按独立节点查看上传、下载、速率和连接数"
      },
      en: {
        label: "Traffic Audit",
        eyebrow: "Per-node Samples",
        title: "Traffic Audit",
        description: "Inspect upload, download, rate, and connections for each generated node"
      }
    }
  },
  {
    id: "routing",
    copy: {
      zh: {
        label: "路由分流",
        eyebrow: "Routing",
        title: "高级路由与主机调优",
        description: "配置 GeoIP、GeoSite、广告过滤和协议分流"
      },
      en: {
        label: "Routing",
        eyebrow: "Routing",
        title: "Advanced Routing and Host Tuning",
        description: "Configure GeoIP, GeoSite, ad blocking, and protocol split rules"
      }
    }
  },
  {
    id: "ha",
    copy: {
      zh: {
        label: "高可用",
        eyebrow: "HA",
        title: "负载均衡与高可用",
        description: "按延迟、丢包和权重选择统一入口后端"
      },
      en: {
        label: "High Availability",
        eyebrow: "HA",
        title: "Load Balancing and High Availability",
        description: "Select unified-entry backends by latency, packet loss, and weight"
      }
    }
  },
  {
    id: "operations",
    copy: {
      zh: {
        label: "告警订阅",
        eyebrow: "自动化运维",
        title: "告警与外部订阅",
        description: "管理 Webhook、Telegram、Server 酱和订阅聚合"
      },
      en: {
        label: "Alerts and Subs",
        eyebrow: "Automation",
        title: "Alerts and External Subscriptions",
        description: "Manage webhooks, Telegram, ServerChan, and subscription aggregation"
      }
    }
  },
  {
    id: "clash",
    copy: {
      zh: {
        label: "Clash 托管",
        eyebrow: "Clash YAML",
        title: "Clash 规则托管",
        description: "维护 Rule Provider、Proxy Group 和托管 YAML"
      },
      en: {
        label: "Clash Hosting",
        eyebrow: "Clash YAML",
        title: "Clash Rule Hosting",
        description: "Maintain Rule Providers, Proxy Groups, and hosted YAML profiles"
      }
    }
  },
  {
    id: "tenants",
    copy: {
      zh: {
        label: "多租户配置",
        eyebrow: "RBAC",
        title: "多租户与配额隔离",
        description: "配置租户、子账号、节点访问权和流量配额"
      },
      en: {
        label: "Tenants",
        eyebrow: "RBAC",
        title: "Multi-tenant Quotas",
        description: "Configure tenants, sub-users, node access, and traffic quotas"
      }
    }
  },
  {
    id: "integrations",
    copy: {
      zh: {
        label: "API 与 Copilot",
        eyebrow: "开放集成",
        title: "REST API 与 AI 运维 Copilot",
        description: "签发 API Key，并让 Copilot 分析异常与日志特征"
      },
      en: {
        label: "API and Copilot",
        eyebrow: "Integrations",
        title: "REST API and AI Operations Copilot",
        description: "Issue API keys and let Copilot inspect incidents and log signals"
      }
    }
  }
];

type ShellProps = {
  activeWorkspace: WorkspaceId;
  children: ReactNode;
  language: Locale;
  onLogout?: () => void;
  onLanguageChange: (language: Locale) => void;
  onThemeChange: (theme: ThemeMode) => void;
  onWorkspaceChange: (workspace: WorkspaceId) => void;
  theme: ThemeMode;
  user?: SessionUser | null;
};

export function Shell({
  activeWorkspace,
  children,
  language,
  onLogout,
  onLanguageChange,
  onThemeChange,
  onWorkspaceChange,
  theme,
  user
}: ShellProps) {
  const active = workspaceItems.find((item) => item.id === activeWorkspace) ?? workspaceItems[0];
  const activeCopy = active.copy[language];

  return (
    <div className="app-shell">
      <aside className="sidebar" aria-label="主导航">
        <div className="brand">
          <div className="brand-mark">OU</div>
          <div>
            <strong>OU-UI</strong>
            <span>{language === "zh" ? "节点运维控制台" : "Node Ops Console"}</span>
          </div>
        </div>
        <nav className="nav-list" aria-label="工作区切换">
          {workspaceItems.map((item) => (
            <button
              aria-current={item.id === activeWorkspace ? "page" : undefined}
              className={item.id === activeWorkspace ? "nav-item active" : "nav-item"}
              key={item.id}
              onClick={() => onWorkspaceChange(item.id)}
              type="button"
            >
              <span className="nav-dot" />
              {item.copy[language].label}
            </button>
          ))}
        </nav>
        <div className="sidebar-footer">
          <span>v4.0.0</span>
          <strong>{language === "zh" ? "控制面在线" : "Control plane online"}</strong>
        </div>
      </aside>
      <main className="workspace">
        <header className="topbar">
          <div className="topbar-title">
            <p className="eyebrow">{activeCopy.eyebrow}</p>
            <h1>{activeCopy.title}</h1>
            <span>{activeCopy.description}</span>
          </div>
          <div className="topbar-actions">
            <div className="mode-switch" aria-label="语言与主题">
              <button
                className={language === "zh" ? "selected" : ""}
                onClick={() => onLanguageChange("zh")}
                type="button"
              >
                {language === "zh" ? "中" : "ZH"}
              </button>
              <button
                className={language === "en" ? "selected" : ""}
                onClick={() => onLanguageChange("en")}
                type="button"
              >
                EN
              </button>
              <button
                aria-label={
                  language === "zh"
                    ? theme === "dark"
                      ? "切换白天模式"
                      : "切换黑夜模式"
                    : theme === "dark"
                      ? "Switch to light mode"
                      : "Switch to dark mode"
                }
                className="theme-toggle"
                onClick={() => onThemeChange(theme === "dark" ? "light" : "dark")}
                type="button"
              >
                {language === "zh" ? (theme === "dark" ? "日" : "夜") : theme === "dark" ? "Light" : "Dark"}
              </button>
            </div>
            {user ? (
              <div className="user-chip" title={user.tenantId || "主租户"}>
                <strong>{user.username}</strong>
                <span>{user.role}</span>
              </div>
            ) : null}
            <label className="search">
              <span>{language === "zh" ? "搜索" : "Search"}</span>
              <input placeholder={language === "zh" ? "Agent、节点、任务或租户" : "Agent, node, task, or tenant"} />
            </label>
            <button className="icon-button" aria-label={language === "zh" ? "通知" : "Notifications"}>
              !
            </button>
            <button className="ghost-button" onClick={onLogout} type="button">
              {language === "zh" ? "退出" : "Sign out"}
            </button>
          </div>
        </header>
        <div className="workspace-body">{children}</div>
      </main>
    </div>
  );
}

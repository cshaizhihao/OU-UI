import type { ReactNode } from "react";
import type { SessionUser } from "../api";

const navItems = [
  { label: "Overview", href: "#overview" },
  { label: "Agents", href: "#agents" },
  { label: "Host Tuning", href: "#agents" },
  { label: "Service Control", href: "#deploy" },
  { label: "Routing", href: "#routing" },
  { label: "HA", href: "#ha" },
  { label: "Alerts", href: "#alerts" },
  { label: "Subscriptions", href: "#subscriptions" },
  { label: "Clash", href: "#clash" },
  { label: "RBAC", href: "#rbac" },
  { label: "API", href: "#api" },
  { label: "Copilot", href: "#copilot" },
  { label: "Queue", href: "#queue" },
  { label: "Metrics", href: "#metrics" },
  { label: "Nodes", href: "#nodes" }
];

type ShellProps = {
  children: ReactNode;
  onLogout?: () => void;
  user?: SessionUser | null;
};

export function Shell({ children, onLogout, user }: ShellProps) {
  return (
    <div className="app-shell">
      <aside className="sidebar" aria-label="Main navigation">
        <div className="brand">
          <div className="brand-mark">OU</div>
          <div>
            <strong>OU-UI</strong>
            <span>Proxy node management</span>
          </div>
        </div>
        <nav className="nav-list">
          {navItems.map((item, index) => (
            <a className={index === 0 ? "nav-item active" : "nav-item"} href={item.href} key={item.label}>
              <span className="nav-dot" />
              {item.label}
            </a>
          ))}
        </nav>
        <div className="sidebar-footer">
          <span>v3.0.0</span>
          <strong>Control plane online</strong>
        </div>
      </aside>
      <main className="workspace">
        <header className="topbar">
          <div>
            <p className="eyebrow">Production Control Plane</p>
            <h1>OU-UI Proxy Node Management Console</h1>
          </div>
          <div className="topbar-actions">
            {user ? (
              <div className="user-chip" title={user.tenantId || "root tenant"}>
                <strong>{user.username}</strong>
                <span>{user.role}</span>
              </div>
            ) : null}
            <label className="search">
              <span>Search</span>
              <input placeholder="Agent, protocol, node, or task ID" />
            </label>
            <button className="icon-button" aria-label="Notifications">
              !
            </button>
            <button className="primary-button">New delivery</button>
            <button className="ghost-button" onClick={onLogout} type="button">
              Sign out
            </button>
          </div>
        </header>
        {children}
      </main>
    </div>
  );
}

import type { ReactNode } from "react";

const navItems = [
  { label: "Overview", href: "#overview" },
  { label: "Agents", href: "#agents" },
  { label: "Runtime Apply", href: "#deploy" },
  { label: "Queue", href: "#queue" },
  { label: "Metrics", href: "#metrics" },
  { label: "Nodes", href: "#nodes" }
];

type ShellProps = {
  children: ReactNode;
};

export function Shell({ children }: ShellProps) {
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
            <a className={index === 0 ? "nav-item active" : "nav-item"} href={item.href} key={item.href}>
              <span className="nav-dot" />
              {item.label}
            </a>
          ))}
        </nav>
        <div className="sidebar-footer">
          <span>v0.5.0</span>
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
            <label className="search">
              <span>Search</span>
              <input placeholder="Agent, protocol, node, or task ID" />
            </label>
            <button className="icon-button" aria-label="Notifications">
              !
            </button>
            <button className="primary-button">New delivery</button>
          </div>
        </header>
        {children}
      </main>
    </div>
  );
}

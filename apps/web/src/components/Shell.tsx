import type { ReactNode } from "react";

const navItems = ["概览", "Agents", "任务", "节点", "指标", "设置"];

type ShellProps = {
  children: ReactNode;
};

export function Shell({ children }: ShellProps) {
  return (
    <div className="app-shell">
      <aside className="sidebar" aria-label="主导航">
        <div className="brand">
          <div className="brand-mark">OU</div>
          <div>
            <strong>OU-UI</strong>
            <span>Agent Ops</span>
          </div>
        </div>
        <nav className="nav-list">
          {navItems.map((item) => (
            <a className={item === "概览" ? "nav-item active" : "nav-item"} href={`#${item}`} key={item}>
              <span className="nav-dot" />
              {item}
            </a>
          ))}
        </nav>
        <div className="sidebar-footer">
          <span>v0.1.0</span>
          <strong>控制台骨架</strong>
        </div>
      </aside>
      <main className="workspace">
        <header className="topbar">
          <div>
            <p className="eyebrow">生产环境</p>
            <h1>Agent 控制台</h1>
          </div>
          <div className="topbar-actions">
            <label className="search">
              <span>搜索</span>
              <input placeholder="Agent、任务或节点" />
            </label>
            <button className="icon-button" aria-label="通知">
              !
            </button>
            <button className="primary-button">新建任务</button>
          </div>
        </header>
        {children}
      </main>
    </div>
  );
}

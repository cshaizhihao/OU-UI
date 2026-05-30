import { FormEvent, useState } from "react";

type LoginPageProps = {
  loading?: boolean;
  error?: string;
  onLogin: (username: string, password: string) => Promise<void>;
};

export function LoginPage({ loading = false, error = "", onLogin }: LoginPageProps) {
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("");

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await onLogin(username, password);
  }

  return (
    <section className="login-page">
      <div className="login-panel">
        <div className="brand large">
          <div className="brand-mark">OU</div>
          <div>
            <strong>OU-UI</strong>
            <span>节点运维控制台</span>
          </div>
        </div>
        <form aria-busy={loading} className="login-form" onSubmit={handleSubmit}>
          <label>
            用户名
            <input
              autoComplete="username"
              disabled={loading}
              onChange={(event) => setUsername(event.target.value)}
              placeholder="admin"
              value={username}
            />
          </label>
          <label>
            密码
            <input
              autoComplete="current-password"
              disabled={loading}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="输入密码"
              type="password"
              value={password}
            />
          </label>
          {error ? <p className="login-error">{error}</p> : null}
          <div className="form-row">
            <label className="checkline">
              <input type="checkbox" defaultChecked disabled={loading} />
              保持登录
            </label>
            <a href="#reset">重置密码</a>
          </div>
          <button className="primary-button" disabled={loading} type="submit">
            {loading ? "登录中" : "登录"}
          </button>
        </form>
      </div>
      <div className="login-context">
        <p className="eyebrow">安全工作区</p>
        <h2>管理 Agent、运行时、协议与自动化任务队列。</h2>
        <div className="context-metrics">
          <span>Xray / Hysteria2</span>
          <span>5 类代理协议</span>
          <span>实时链路监控</span>
        </div>
      </div>
    </section>
  );
}

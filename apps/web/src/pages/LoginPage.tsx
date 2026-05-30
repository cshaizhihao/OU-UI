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
            <span>Node Control Console</span>
          </div>
        </div>
        <form aria-busy={loading} className="login-form" onSubmit={handleSubmit}>
          <label>
            Username
            <input
              autoComplete="username"
              disabled={loading}
              onChange={(event) => setUsername(event.target.value)}
              placeholder="admin"
              value={username}
            />
          </label>
          <label>
            Password
            <input
              autoComplete="current-password"
              disabled={loading}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="Enter password"
              type="password"
              value={password}
            />
          </label>
          {error ? <p className="login-error">{error}</p> : null}
          <div className="form-row">
            <label className="checkline">
              <input type="checkbox" defaultChecked disabled={loading} />
              Keep signed in
            </label>
            <a href="#reset">Reset password</a>
          </div>
          <button className="primary-button" disabled={loading} type="submit">
            {loading ? "Signing in" : "Sign in"}
          </button>
        </form>
      </div>
      <div className="login-context">
        <p className="eyebrow">Secure Workspace</p>
        <h2>Manage agents, runtimes, protocols, and delivery queues.</h2>
        <div className="context-metrics">
          <span>Xray / Hysteria2</span>
          <span>5 proxy protocols</span>
          <span>Live link monitoring</span>
        </div>
      </div>
    </section>
  );
}

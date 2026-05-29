export function LoginPage() {
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
        <form className="login-form">
          <label>
            Work email
            <input type="email" placeholder="name@company.com" />
          </label>
          <label>
            Password
            <input type="password" placeholder="Enter password" />
          </label>
          <div className="form-row">
            <label className="checkline">
              <input type="checkbox" defaultChecked />
              Keep signed in
            </label>
            <a href="#reset">Reset password</a>
          </div>
          <button className="primary-button" type="button">
            Sign in
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

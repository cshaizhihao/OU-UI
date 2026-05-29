export function LoginPage() {
  return (
    <section className="login-page">
      <div className="login-panel">
        <div className="brand large">
          <div className="brand-mark">OU</div>
          <div>
            <strong>OU-UI</strong>
            <span>Agent Ops Console</span>
          </div>
        </div>
        <form className="login-form">
          <label>
            工作邮箱
            <input type="email" placeholder="name@company.com" />
          </label>
          <label>
            密码
            <input type="password" placeholder="输入密码" />
          </label>
          <div className="form-row">
            <label className="checkline">
              <input type="checkbox" defaultChecked />
              保持登录
            </label>
            <a href="#reset">忘记密码</a>
          </div>
          <button className="primary-button" type="button">
            登录控制台
          </button>
        </form>
      </div>
      <div className="login-context">
        <p className="eyebrow">Secure Workspace</p>
        <h2>统一查看 Agent、任务与节点状态。</h2>
        <div className="context-metrics">
          <span>99.98% 可用性</span>
          <span>42 个自动化流</span>
          <span>8 个生产节点</span>
        </div>
      </div>
    </section>
  );
}

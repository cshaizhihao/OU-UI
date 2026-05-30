export function WorkspaceSkeleton({ title = "工作区加载中" }: { title?: string }) {
  return (
    <div className="workspace-view" aria-busy="true">
      <section className="panel skeleton-panel">
        <div className="view-heading skeleton-heading">
          <div className="skeleton-stack">
            <span className="skeleton-line tiny" />
            <span className="skeleton-line xlarge" />
            <span className="skeleton-line medium" />
          </div>
          <span className="skeleton-block button" />
        </div>
        <span className="sr-only">{title}</span>
      </section>

      <section className="kpi-grid skeleton-kpi-grid" aria-label="指标加载中">
        {Array.from({ length: 4 }).map((_, index) => (
          <article className="kpi-card skeleton-card" key={index}>
            <span className="skeleton-line tiny" />
            <span className="skeleton-line large" />
            <span className="skeleton-line small" />
          </article>
        ))}
      </section>

      <div className="workspace-grid two">
        {Array.from({ length: 2 }).map((_, panelIndex) => (
          <section className="panel skeleton-panel" key={panelIndex}>
            <div className="section-heading">
              <div className="skeleton-stack">
                <span className="skeleton-line tiny" />
                <span className="skeleton-line large" />
              </div>
              <span className="skeleton-block button" />
            </div>
            <div className="skeleton-table">
              {Array.from({ length: 5 }).map((__, rowIndex) => (
                <span className="skeleton-line wide" key={rowIndex} />
              ))}
            </div>
          </section>
        ))}
      </div>
    </div>
  );
}

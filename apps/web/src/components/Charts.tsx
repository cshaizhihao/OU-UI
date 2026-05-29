const bars = [62, 78, 54, 88, 71, 96, 83, 69, 91, 74, 87, 80];
const nodes = [38, 55, 48, 72, 61, 83, 76, 88, 81, 93];

export function AnalyticsPanel() {
  return (
    <section className="panel chart-panel" id="指标">
      <div className="section-heading">
        <div>
          <p className="eyebrow">Observability</p>
          <h2>实时吞吐趋势</h2>
        </div>
        <select aria-label="时间范围">
          <option>最近 12 小时</option>
          <option>最近 24 小时</option>
          <option>最近 7 天</option>
        </select>
      </div>
      <div className="chart-grid">
        <div className="bar-chart" aria-label="吞吐柱状图占位">
          {bars.map((height, index) => (
            <span key={index} style={{ height: `${height}%` }} />
          ))}
        </div>
        <div className="line-card">
          <svg viewBox="0 0 320 150" role="img" aria-label="节点负载折线图占位">
            <defs>
              <linearGradient id="lineFill" x1="0" x2="0" y1="0" y2="1">
                <stop offset="0%" stopColor="#2563eb" stopOpacity="0.26" />
                <stop offset="100%" stopColor="#2563eb" stopOpacity="0" />
              </linearGradient>
            </defs>
            <path
              d="M8 132 L40 108 L74 116 L108 80 L142 94 L176 56 L210 66 L244 36 L278 47 L312 22"
              fill="none"
              stroke="#2563eb"
              strokeLinecap="round"
              strokeWidth="4"
            />
            <path
              d="M8 132 L40 108 L74 116 L108 80 L142 94 L176 56 L210 66 L244 36 L278 47 L312 22 L312 150 L8 150 Z"
              fill="url(#lineFill)"
            />
          </svg>
          <div className="spark-row">
            {nodes.map((value, index) => (
              <span key={index} style={{ height: `${value}%` }} />
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}

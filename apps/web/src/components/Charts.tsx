const uplinkBars = [52, 68, 44, 76, 61, 88, 73, 58, 82, 64, 79, 70];
const queueBars = [28, 35, 31, 44, 39, 53, 48, 56, 51, 62];

export function AnalyticsPanel() {
  return (
    <section className="panel chart-panel" id="metrics">
      <div className="section-heading">
        <div>
          <p className="eyebrow">Observability</p>
          <h2>Link throughput and queue trend</h2>
        </div>
        <select aria-label="Time range">
          <option>Last 12 hours</option>
          <option>Last 24 hours</option>
          <option>Last 7 days</option>
        </select>
      </div>
      <div className="chart-grid">
        <div className="bar-chart" aria-label="Uplink throughput bar chart">
          {uplinkBars.map((height, index) => (
            <span key={index} style={{ height: `${height}%` }} />
          ))}
        </div>
        <div className="line-card">
          <svg viewBox="0 0 320 150" role="img" aria-label="Queue trend line chart">
            <defs>
              <linearGradient id="lineFill" x1="0" x2="0" y1="0" y2="1">
                <stop offset="0%" stopColor="#0891b2" stopOpacity="0.26" />
                <stop offset="100%" stopColor="#0891b2" stopOpacity="0" />
              </linearGradient>
            </defs>
            <path
              d="M8 122 L40 104 L74 110 L108 82 L142 92 L176 64 L210 70 L244 44 L278 54 L312 30"
              fill="none"
              stroke="#0891b2"
              strokeLinecap="round"
              strokeWidth="4"
            />
            <path
              d="M8 122 L40 104 L74 110 L108 82 L142 92 L176 64 L210 70 L244 44 L278 54 L312 30 L312 150 L8 150 Z"
              fill="url(#lineFill)"
            />
          </svg>
          <div className="spark-row">
            {queueBars.map((value, index) => (
              <span key={index} style={{ height: `${value}%` }} />
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}

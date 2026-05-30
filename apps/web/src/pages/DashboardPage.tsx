import { useEffect, useState } from "react";
import type { DashboardDTO } from "../api";
import { ClashWorkspace } from "../components/ClashWorkspace";
import { HAWorkspace } from "../components/HAWorkspace";
import { IntegrationsWorkspace } from "../components/IntegrationsWorkspace";
import { NodeManagementWorkspace } from "../components/NodeManagementWorkspace";
import { NoticeRow } from "../components/ConsolePrimitives";
import { OperationsWorkspace } from "../components/OperationsWorkspace";
import { OverviewWorkspace } from "../components/OverviewWorkspace";
import { RoutingWorkspace } from "../components/RoutingWorkspace";
import { TenantWorkspace } from "../components/TenantWorkspace";
import { TrafficAuditWorkspace } from "../components/TrafficAuditWorkspace";
import type { WorkspaceId } from "../components/Shell";
import { WorkspaceSkeleton } from "../components/WorkspaceSkeleton";

type DashboardPageProps = {
  activeWorkspace: WorkspaceId;
  data: DashboardDTO | null;
  loading?: boolean;
  error?: string;
  onRefresh?: () => void;
};

export function DashboardPage({
  activeWorkspace,
  data,
  loading = false,
  error = "",
  onRefresh
}: DashboardPageProps) {
  const [switching, setSwitching] = useState(false);
  const agents = data?.agents ?? [];
  const isInitialLoading = loading && !data;
  const isRefreshing = loading && Boolean(data);

  useEffect(() => {
    setSwitching(true);
    const timer = window.setTimeout(() => setSwitching(false), 180);
    return () => window.clearTimeout(timer);
  }, [activeWorkspace]);

  if (isInitialLoading || switching) {
    return (
      <div className="dashboard" aria-busy="true">
        <WorkspaceSkeleton title="工作区加载中" />
      </div>
    );
  }

  return (
    <div className={`dashboard console-dashboard${isRefreshing ? " is-refreshing" : ""}`} aria-busy={loading}>
      {isRefreshing ? <span className="refresh-rail" aria-hidden="true" /> : null}
      {error ? (
        <NoticeRow tone="danger">
          <strong>{error}</strong>
          <button className="ghost-button" onClick={onRefresh} type="button">
            重试
          </button>
        </NoticeRow>
      ) : null}
      <div className="workspace-transition" key={activeWorkspace}>
        {renderWorkspace(activeWorkspace, data, agents, loading, onRefresh)}
      </div>
    </div>
  );
}

function renderWorkspace(
  activeWorkspace: WorkspaceId,
  data: DashboardDTO | null,
  agents: DashboardDTO["agents"],
  loading: boolean,
  onRefresh?: () => void
) {
  switch (activeWorkspace) {
    case "nodes":
      return <NodeManagementWorkspace agents={agents} data={data} />;
    case "traffic":
      return <TrafficAuditWorkspace agents={agents} data={data} />;
    case "routing":
      return <RoutingWorkspace agents={agents} data={data} disabled={loading} onRefresh={onRefresh} />;
    case "ha":
      return <HAWorkspace agents={agents} data={data} disabled={loading} onRefresh={onRefresh} />;
    case "operations":
      return <OperationsWorkspace data={data} disabled={loading} onRefresh={onRefresh} />;
    case "clash":
      return <ClashWorkspace data={data} disabled={loading} onRefresh={onRefresh} />;
    case "tenants":
      return <TenantWorkspace data={data} disabled={loading} onRefresh={onRefresh} />;
    case "integrations":
      return <IntegrationsWorkspace data={data} disabled={loading} onRefresh={onRefresh} />;
    case "overview":
    default:
      return <OverviewWorkspace agents={agents} data={data} onRefresh={onRefresh} />;
  }
}

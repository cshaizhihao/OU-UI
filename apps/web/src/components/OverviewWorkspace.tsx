import type { DashboardDTO } from "../api";
import type { Agent } from "../data";
import { OnboardingLaunchpad } from "./OnboardingLaunchpad";

type OverviewWorkspaceProps = {
  agents: Agent[];
  data: DashboardDTO | null;
  onRefresh?: () => void;
};

export function OverviewWorkspace({ agents, data, onRefresh }: OverviewWorkspaceProps) {
  return (
    <div className="workspace-view">
      <OnboardingLaunchpad agents={agents} data={data} onRefresh={onRefresh} />
    </div>
  );
}

import {
  primaryWorkspaceId,
  professionalWorkspaceIds,
  starterStepIds,
  starterStepDefinitions,
  workspaceModeLabel
} from "./onboarding";

const expectedStarterFlow = [
  "connect-agent",
  "create-node",
  "copy-link",
  "verify-status"
] as const;

const expectedProfessionalWorkspaces = [
  "nodes",
  "traffic",
  "routing",
  "ha",
  "operations",
  "clash",
  "tenants",
  "integrations"
] as const;

type Equal<A, B> =
  (<T>() => T extends A ? 1 : 2) extends
  (<T>() => T extends B ? 1 : 2) ? true : false;

type Expect<T extends true> = T;

type _PrimaryWorkspaceIsOverview = Expect<Equal<typeof primaryWorkspaceId, "overview">>;
type _StarterFlowIsExactlyFourSteps = Expect<Equal<typeof starterStepIds, typeof expectedStarterFlow>>;
type _StarterDefinitionsUseOnlyStarterIds = Expect<Equal<typeof starterStepDefinitions[number]["id"], typeof expectedStarterFlow[number]>>;
type _ProfessionalWorkspacesAreHiddenGroup = Expect<Equal<typeof professionalWorkspaceIds, typeof expectedProfessionalWorkspaces>>;
type _ModeLabelHasChineseDefault = Expect<Equal<typeof workspaceModeLabel.zh, "专业模式">>;

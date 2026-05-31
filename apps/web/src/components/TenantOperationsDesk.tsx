import { useMemo, useState } from "react";
import type { DashboardDTO, NodeTraffic, PanelUser, Tenant } from "../api";
import { formatBytes, KpiGrid, SectionHeader, StatusTag, useLocale } from "./ConsolePrimitives";

type TenantOperationsDeskProps = {
  data: DashboardDTO | null;
};

type AccessUsage = {
  connections: number;
  peakNodeBytes: number;
  totalBytes: number;
};

type TenantRisk = "danger" | "muted" | "ok" | "warning";

export function TenantOperationsDesk({ data }: TenantOperationsDeskProps) {
  const tenants = data?.control.tenants ?? [];
  const users = data?.control.users ?? [];
  const traffic = data?.control.traffic ?? [];
  const [selectedTenantId, setSelectedTenantId] = useState("");
  const language = useLocale();
  const tenantLookup = useMemo(() => new Map(tenants.map((tenant) => [tenant.id, tenant])), [tenants]);
  const selectedTenant = tenants.find((tenant) => tenant.id === selectedTenantId) ?? tenants[0];
  const selectedUsers = selectedTenant ? users.filter((user) => user.tenantId === selectedTenant.id) : [];
  const summary = buildTenantSummary(tenants, users, traffic);

  return (
    <>
      <KpiGrid
        items={[
          { label: "租户总数", value: String(tenants.length), delta: formatScopedCount(summary.scopedTenants, "个已限权", language) },
          { label: "子账号", value: String(users.length), delta: formatScopedCount(summary.scopedUsers, "个独立范围", language) },
          { label: "配额压力", value: `${summary.highestPressure}%`, delta: riskLabel(summary.risk, language) },
          { label: "实时连接占用", value: String(summary.connections), delta: formatBytes(summary.usedBytes) }
        ]}
      />

      <div className="tenant-ops-grid">
        <section className="panel tenant-ledger-panel">
          <SectionHeader eyebrow="租户运营台" title="租户访问矩阵" />
          <div className="tenant-ledger">
            {tenants.map((tenant) => (
              <TenantCard
                key={tenant.id}
                active={(selectedTenant?.id ?? "") === tenant.id}
                onSelect={() => setSelectedTenantId(tenant.id)}
                tenant={tenant}
                usage={usageForAccess(tenant.nodeAccess ?? [], traffic)}
              />
            ))}
            {tenants.length === 0 ? <p className="empty-state">暂无租户</p> : null}
          </div>
        </section>

        <section className="panel tenant-detail-panel">
          <SectionHeader eyebrow="租户画像" title={selectedTenant?.name ?? "等待租户接入"} />
          {selectedTenant ? (
            <TenantDetail tenant={selectedTenant} users={selectedUsers} usage={usageForAccess(selectedTenant.nodeAccess ?? [], traffic)} />
          ) : (
            <p className="empty-state">创建租户后会显示配额、访问范围和子账号继承关系</p>
          )}
        </section>
      </div>

      <section className="panel">
        <SectionHeader eyebrow="账号运营" title="子账号访问看板" />
        <div className="tenant-user-card-grid">
          {users.map((user) => (
            <UserAccessCard key={user.id} tenant={tenantLookup.get(user.tenantId ?? "")} traffic={traffic} user={user} />
          ))}
          {users.length === 0 ? <p className="empty-state">暂无子账号</p> : null}
        </div>
      </section>
    </>
  );
}

function TenantCard({
  active,
  onSelect,
  tenant,
  usage
}: {
  active: boolean;
  onSelect: () => void;
  tenant: Tenant;
  usage: AccessUsage;
}) {
  const language = useLocale();
  const risk = tenantRisk(usage, tenant.monthlyTrafficQuota, tenant.perNodeTrafficQuota, tenant.maxConnections);
  return (
    <button className={active ? "tenant-card selected" : "tenant-card"} onClick={onSelect} type="button">
      <div className="tenant-card-head">
        <div>
          <strong>{tenant.name}</strong>
          <span>{tenant.id}</span>
        </div>
        <StatusTag tone={statusTone(tenant.status)}>{statusLabel(tenant.status, language)}</StatusTag>
      </div>
      <ScopeChips access={tenant.nodeAccess ?? []} />
      <div className="tenant-meter-stack">
        <QuotaMeter label="月度配额" quota={tenant.monthlyTrafficQuota ?? 0} used={usage.totalBytes} />
        <QuotaMeter label="单节点配额" quota={tenant.perNodeTrafficQuota ?? 0} used={usage.peakNodeBytes} />
      </div>
      <div className="tenant-card-stats">
        <span>{formatConnectionUsage(usage.connections, tenant.maxConnections, language)}</span>
        <span className={`tenant-risk tenant-risk-${risk}`}>{riskLabel(risk, language)}</span>
      </div>
    </button>
  );
}

function TenantDetail({ tenant, usage, users }: { tenant: Tenant; usage: AccessUsage; users: PanelUser[] }) {
  const language = useLocale();
  return (
    <div className="tenant-detail-body">
      <div className="tenant-detail-grid">
        <MetricBox label="月度用量" value={formatQuotaPair(usage.totalBytes, tenant.monthlyTrafficQuota ?? 0, language)} />
        <MetricBox label="峰值单节点" value={formatQuotaPair(usage.peakNodeBytes, tenant.perNodeTrafficQuota ?? 0, language)} />
        <MetricBox label="连接占用" value={formatConnectionUsage(usage.connections, tenant.maxConnections, language)} />
        <MetricBox label="子账号" value={String(users.length)} />
      </div>
      <div className="tenant-policy-line">
        <span>节点访问</span>
        <ScopeChips access={tenant.nodeAccess ?? []} />
      </div>
      <div className="tenant-user-strip">
        {users.slice(0, 5).map((user) => (
          <span key={user.id}>
            <strong>{user.username}</strong>
            {user.role}
          </span>
        ))}
        {users.length === 0 ? <span>暂无子账号</span> : null}
      </div>
    </div>
  );
}

function UserAccessCard({ tenant, traffic, user }: { tenant?: Tenant; traffic: NodeTraffic[]; user: PanelUser }) {
  const language = useLocale();
  const access = effectiveUserAccess(user, tenant);
  const usage = usageForAccess(access, traffic);
  const risk = tenantRisk(usage, user.monthlyTrafficQuota, user.perNodeTrafficQuota, user.maxConnections);
  return (
    <article className="tenant-user-card">
      <div className="tenant-card-head">
        <div>
          <strong>{user.username}</strong>
          <span>{tenant?.name ?? (language === "zh-CN" ? "主租户" : "Root tenant")}</span>
        </div>
        <StatusTag tone={statusTone(user.status)}>{statusLabel(user.status, language)}</StatusTag>
      </div>
      <ScopeChips access={access} />
      <div className="tenant-detail-grid compact">
        <MetricBox label="角色" value={user.role} />
        <MetricBox label="月度配额" value={formatQuotaPair(usage.totalBytes, user.monthlyTrafficQuota ?? 0, language)} />
        <MetricBox label="单节点配额" value={formatQuotaPair(usage.peakNodeBytes, user.perNodeTrafficQuota ?? 0, language)} />
        <MetricBox label="连接数" value={formatConnectionUsage(usage.connections, user.maxConnections, language)} />
      </div>
      <span className={`tenant-risk tenant-risk-${risk}`}>{riskLabel(risk, language)}</span>
    </article>
  );
}

function QuotaMeter({ label, quota, used }: { label: string; quota: number; used: number }) {
  const language = useLocale();
  const percent = quotaPercent(used, quota);
  return (
    <div className="tenant-meter">
      <div>
        <span>{label}</span>
        <strong>{formatQuotaPair(used, quota, language)}</strong>
      </div>
      <div className="tenant-meter-track">
        <span style={{ width: `${percent}%` }} />
      </div>
    </div>
  );
}

function MetricBox({ label, value }: { label: string; value: string }) {
  return (
    <div className="tenant-metric-box">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function ScopeChips({ access }: { access: string[] }) {
  const language = useLocale();
  const visible = normalizedAccess(access);
  const scope = visible.length === 0 || visible.includes("*") ? [language === "zh-CN" ? "全部节点" : "All nodes"] : visible.slice(0, 4);
  return (
    <div className="tenant-scope-chips">
      {scope.map((item) => (
        <span key={item}>{item}</span>
      ))}
      {visible.length > 4 && !visible.includes("*") ? <span>+{visible.length - 4}</span> : null}
    </div>
  );
}

function buildTenantSummary(tenants: Tenant[], users: PanelUser[], traffic: NodeTraffic[]): {
  connections: number;
  highestPressure: number;
  risk: TenantRisk;
  scopedTenants: number;
  scopedUsers: number;
  usedBytes: number;
} {
  const allUsage = usageForAccess([], traffic);
  const tenantRisks = tenants.map((tenant) =>
    tenantRisk(usageForAccess(tenant.nodeAccess ?? [], traffic), tenant.monthlyTrafficQuota, tenant.perNodeTrafficQuota, tenant.maxConnections)
  );
  return {
    connections: allUsage.connections,
    highestPressure: Math.max(
      0,
      ...tenants.map((tenant) => quotaPercent(usageForAccess(tenant.nodeAccess ?? [], traffic).totalBytes, tenant.monthlyTrafficQuota ?? 0))
    ),
    risk: tenantRisks.includes("danger") ? "danger" : tenantRisks.includes("warning") ? "warning" : tenants.length ? "ok" : "muted",
    scopedTenants: tenants.filter((tenant) => !isWildcardAccess(tenant.nodeAccess ?? [])).length,
    scopedUsers: users.filter((user) => !isWildcardAccess(user.nodeAccess ?? [])).length,
    usedBytes: allUsage.totalBytes
  };
}

function usageForAccess(access: string[], traffic: NodeTraffic[]): AccessUsage {
  const normalized = normalizedAccess(access);
  const latest = [...traffic]
    .sort((a, b) => new Date(b.collectedAt).getTime() - new Date(a.collectedAt).getTime())
    .filter((sample, index, samples) => samples.findIndex((item) => item.nodeId === sample.nodeId) === index)
    .filter((sample) => accessMatches(sample, normalized));

  const perNode = latest.map((sample) => sample.rxBytes + sample.txBytes);
  return {
    connections: latest.reduce((total, sample) => total + sample.connections, 0),
    peakNodeBytes: perNode.length ? Math.max(...perNode) : 0,
    totalBytes: perNode.reduce((total, value) => total + value, 0)
  };
}

function accessMatches(sample: NodeTraffic, access: string[]): boolean {
  if (isWildcardAccess(access)) {
    return true;
  }
  return access.includes(sample.nodeId) || access.includes(sample.agentId);
}

function effectiveUserAccess(user: PanelUser, tenant?: Tenant): string[] {
  return normalizedAccess(user.nodeAccess ?? []).length ? normalizedAccess(user.nodeAccess ?? []) : normalizedAccess(tenant?.nodeAccess ?? []);
}

function normalizedAccess(access: string[]): string[] {
  return access.map((item) => item.trim()).filter(Boolean);
}

function isWildcardAccess(access: string[]): boolean {
  const normalized = normalizedAccess(access);
  return normalized.length === 0 || normalized.includes("*");
}

function tenantRisk(usage: AccessUsage, monthlyQuota = 0, perNodeQuota = 0, maxConnections = 0): TenantRisk {
  const monthly = quotaPercent(usage.totalBytes, monthlyQuota);
  const perNode = quotaPercent(usage.peakNodeBytes, perNodeQuota);
  const connections = maxConnections > 0 ? Math.min(Math.round((usage.connections / maxConnections) * 100), 100) : 0;
  const highest = Math.max(monthly, perNode, connections);
  if (monthlyQuota <= 0 && perNodeQuota <= 0 && maxConnections <= 0) {
    return "muted";
  }
  if (highest >= 95) {
    return "danger";
  }
  if (highest >= 80) {
    return "warning";
  }
  return "ok";
}

function quotaPercent(used: number, quota: number): number {
  if (!Number.isFinite(quota) || quota <= 0) {
    return 0;
  }
  return Math.min(100, Math.round((used / quota) * 100));
}

function statusTone(status: string): "danger" | "info" | "muted" | "ok" | "warning" {
  if (status === "active" || status === "online") {
    return "ok";
  }
  if (status === "paused" || status === "disabled") {
    return "warning";
  }
  return "muted";
}

function statusLabel(status: string, language: "zh-CN" | "en"): string {
  const normalized = status.toLowerCase();
  if (language === "en") {
    return normalized || "unknown";
  }
  if (normalized === "active") {
    return "启用";
  }
  if (normalized === "paused") {
    return "暂停";
  }
  if (normalized === "disabled") {
    return "停用";
  }
  return status || "未知";
}

function riskLabel(risk: TenantRisk, language: "zh-CN" | "en"): string {
  if (language === "en") {
    return { danger: "Limit reached", muted: "No quota", ok: "Healthy", warning: "Watch quota" }[risk];
  }
  return { danger: "已触顶", muted: "未限额", ok: "健康", warning: "接近配额" }[risk];
}

function formatScopedCount(count: number, suffix: string, language: "zh-CN" | "en"): string {
  if (language === "en") {
    return `${count} scoped`;
  }
  return `${count} ${suffix}`;
}

function formatQuotaPair(used: number, quota: number, language: "zh-CN" | "en"): string {
  return quota > 0 ? `${formatBytes(used)} / ${formatBytes(quota)}` : `${formatBytes(used)} / ${unlimitedLabel(language)}`;
}

function formatConnectionUsage(used: number, quota: number | undefined, language: "zh-CN" | "en"): string {
  const label = quota && quota > 0 ? `${used} / ${quota}` : `${used} / ${unlimitedLabel(language)}`;
  return language === "zh-CN" ? `${label} 连接` : `${label} conn`;
}

function unlimitedLabel(language: "zh-CN" | "en" = "zh-CN"): string {
  return language === "zh-CN" ? "不限制" : "Unlimited";
}

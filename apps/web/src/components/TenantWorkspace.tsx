import { useEffect, useState, type FormEvent } from "react";
import { createPanelUser, createTenant, type DashboardDTO } from "../api";
import {
  gbToBytes,
  NoticeRow,
  parseCSV,
  SectionHeader,
  ViewHeading
} from "./ConsolePrimitives";
import { TenantOperationsDesk } from "./TenantOperationsDesk";

type TenantWorkspaceProps = {
  data: DashboardDTO | null;
  disabled?: boolean;
  onRefresh?: () => void;
};

export function TenantWorkspace({ data, disabled = false, onRefresh }: TenantWorkspaceProps) {
  const firstAgentId = data?.agents[0]?.id ?? "";
  const [busy, setBusy] = useState("");
  const [message, setMessage] = useState("");
  const [tenant, setTenant] = useState({
    name: "运维租户",
    nodeAccess: firstAgentId,
    monthlyTrafficGb: 1024,
    perNodeTrafficGb: 256,
    maxConnections: 2000
  });
  const [panelUser, setPanelUser] = useState({
    username: "operator",
    password: "",
    tenantId: "",
    nodeAccess: firstAgentId,
    monthlyTrafficGb: 256,
    perNodeTrafficGb: 64,
    maxConnections: 500
  });
  const controlsDisabled = disabled || !data;

  useEffect(() => {
    if (!firstAgentId) {
      return;
    }
    setTenant((current) => (current.nodeAccess ? current : { ...current, nodeAccess: firstAgentId }));
    setPanelUser((current) => (current.nodeAccess ? current : { ...current, nodeAccess: firstAgentId }));
  }, [firstAgentId]);

  async function runAction(label: string, action: () => Promise<unknown>) {
    setBusy(label);
    setMessage("");
    try {
      await action();
      setMessage(`${label} 已完成`);
      await Promise.resolve(onRefresh?.());
    } catch (err) {
      setMessage(err instanceof Error ? err.message : `${label} 执行失败`);
    } finally {
      setBusy("");
    }
  }

  function handleCreateTenant(event: FormEvent) {
    event.preventDefault();
    void runAction("租户", () =>
      createTenant({
        name: tenant.name,
        status: "active",
        role: "operator",
        nodeAccess: parseCSV(tenant.nodeAccess),
        monthlyTrafficQuota: gbToBytes(tenant.monthlyTrafficGb),
        perNodeTrafficQuota: gbToBytes(tenant.perNodeTrafficGb),
        maxConnections: Number(tenant.maxConnections) || 0
      })
    );
  }

  function handleCreatePanelUser(event: FormEvent) {
    event.preventDefault();
    void runAction("子账号", () =>
      createPanelUser({
        username: panelUser.username,
        password: panelUser.password,
        tenantId: panelUser.tenantId,
        role: "operator",
        status: "active",
        nodeAccess: parseCSV(panelUser.nodeAccess),
        monthlyTrafficQuota: gbToBytes(panelUser.monthlyTrafficGb),
        perNodeTrafficQuota: gbToBytes(panelUser.perNodeTrafficGb),
        maxConnections: Number(panelUser.maxConnections) || 0
      })
    );
  }

  return (
    <div className="workspace-view">
      <ViewHeading
        description="租户、子账号、节点访问权、月度配额和单节点配额在独立工作区内完成。"
        eyebrow="多租户"
        title="RBAC 与配额隔离"
      />
      {message ? <NoticeRow>{message}</NoticeRow> : null}

      <TenantOperationsDesk data={data} />

      <div className="workspace-grid two">
        <section className="panel">
          <SectionHeader eyebrow="Tenant" title="创建租户" />
          <form className="control-form" onSubmit={handleCreateTenant}>
            <label>
              租户名称
              <input value={tenant.name} onChange={(event) => setTenant({ ...tenant, name: event.target.value })} />
            </label>
            <label>
              节点访问
              <input value={tenant.nodeAccess} onChange={(event) => setTenant({ ...tenant, nodeAccess: event.target.value })} />
            </label>
            <label>
              月度 GB 配额
              <input
                type="number"
                value={tenant.monthlyTrafficGb}
                onChange={(event) => setTenant({ ...tenant, monthlyTrafficGb: Number(event.target.value) })}
              />
            </label>
            <label>
              单节点 GB 配额
              <input
                type="number"
                value={tenant.perNodeTrafficGb}
                onChange={(event) => setTenant({ ...tenant, perNodeTrafficGb: Number(event.target.value) })}
              />
            </label>
            <label>
              最大连接数
              <input
                type="number"
                value={tenant.maxConnections}
                onChange={(event) => setTenant({ ...tenant, maxConnections: Number(event.target.value) })}
              />
            </label>
            <button className="primary-button" disabled={Boolean(busy) || controlsDisabled} type="submit">
              创建租户
            </button>
          </form>
        </section>

        <section className="panel">
          <SectionHeader eyebrow="User" title="创建子账号" />
          <form className="control-form" onSubmit={handleCreatePanelUser}>
            <label>
              用户名
              <input value={panelUser.username} onChange={(event) => setPanelUser({ ...panelUser, username: event.target.value })} />
            </label>
            <label>
              密码
              <input
                minLength={10}
                placeholder="至少 10 位临时密码"
                required
                type="password"
                value={panelUser.password}
                onChange={(event) => setPanelUser({ ...panelUser, password: event.target.value })}
              />
            </label>
            <label>
              租户 ID
              <select value={panelUser.tenantId} onChange={(event) => setPanelUser({ ...panelUser, tenantId: event.target.value })}>
                <option value="">主租户</option>
                {(data?.control.tenants ?? []).map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              节点访问
              <input value={panelUser.nodeAccess} onChange={(event) => setPanelUser({ ...panelUser, nodeAccess: event.target.value })} />
            </label>
            <label>
              月度 GB 配额
              <input
                type="number"
                value={panelUser.monthlyTrafficGb}
                onChange={(event) => setPanelUser({ ...panelUser, monthlyTrafficGb: Number(event.target.value) })}
              />
            </label>
            <label>
              单节点 GB 配额
              <input
                type="number"
                value={panelUser.perNodeTrafficGb}
                onChange={(event) => setPanelUser({ ...panelUser, perNodeTrafficGb: Number(event.target.value) })}
              />
            </label>
            <label>
              最大连接数
              <input
                type="number"
                value={panelUser.maxConnections}
                onChange={(event) => setPanelUser({ ...panelUser, maxConnections: Number(event.target.value) })}
              />
            </label>
            <button className="ghost-button" disabled={Boolean(busy) || controlsDisabled} type="submit">
              创建子账号
            </button>
          </form>
        </section>
      </div>
    </div>
  );
}

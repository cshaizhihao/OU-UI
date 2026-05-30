import { useEffect, useState, type FormEvent } from "react";
import { createPanelUser, createTenant, updatePanelUser, updateTenant, type DashboardDTO, type PanelUser, type Tenant } from "../api";
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
  const [tenantPolicy, setTenantPolicy] = useState({
    id: "",
    name: "",
    status: "active",
    nodeAccess: "",
    monthlyTrafficGb: 0,
    perNodeTrafficGb: 0,
    maxConnections: 0
  });
  const [userPolicy, setUserPolicy] = useState({
    id: "",
    tenantId: "",
    username: "",
    password: "",
    status: "active",
    nodeAccess: "",
    monthlyTrafficGb: 0,
    perNodeTrafficGb: 0,
    maxConnections: 0
  });
  const controlsDisabled = disabled || !data;

  useEffect(() => {
    if (!firstAgentId) {
      return;
    }
    setTenant((current) => (current.nodeAccess ? current : { ...current, nodeAccess: firstAgentId }));
    setPanelUser((current) => (current.nodeAccess ? current : { ...current, nodeAccess: firstAgentId }));
  }, [firstAgentId]);

  useEffect(() => {
    const current = data?.control.tenants.find((item) => item.id === tenantPolicy.id) ?? data?.control.tenants[0];
    if (current && current.id !== tenantPolicy.id) {
      setTenantPolicy(tenantToPolicy(current));
    }
  }, [data?.control.tenants, tenantPolicy.id]);

  useEffect(() => {
    const current = data?.control.users.find((item) => item.id === userPolicy.id) ?? data?.control.users[0];
    if (current && current.id !== userPolicy.id) {
      setUserPolicy(userToPolicy(current));
    }
  }, [data?.control.users, userPolicy.id]);

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

  function handleUpdateTenant(event: FormEvent) {
    event.preventDefault();
    if (!tenantPolicy.id) {
      setMessage("请先选择一个租户");
      return;
    }
    void runAction("租户治理", () =>
      updateTenant(tenantPolicy.id, {
        name: tenantPolicy.name,
        status: tenantPolicy.status,
        role: "operator",
        nodeAccess: parseCSV(tenantPolicy.nodeAccess),
        monthlyTrafficQuota: gbToBytes(tenantPolicy.monthlyTrafficGb),
        perNodeTrafficQuota: gbToBytes(tenantPolicy.perNodeTrafficGb),
        maxConnections: Number(tenantPolicy.maxConnections) || 0
      })
    );
  }

  function handleUpdatePanelUser(event: FormEvent) {
    event.preventDefault();
    if (!userPolicy.id) {
      setMessage("请先选择一个子账号");
      return;
    }
    const payload: Parameters<typeof updatePanelUser>[1] = {
      tenantId: userPolicy.tenantId,
      username: userPolicy.username,
      status: userPolicy.status,
      role: "operator",
      nodeAccess: parseCSV(userPolicy.nodeAccess),
      monthlyTrafficQuota: gbToBytes(userPolicy.monthlyTrafficGb),
      perNodeTrafficQuota: gbToBytes(userPolicy.perNodeTrafficGb),
      maxConnections: Number(userPolicy.maxConnections) || 0
    };
    if (userPolicy.password) {
      payload.password = userPolicy.password;
    }
    void runAction("子账号治理", () => updatePanelUser(userPolicy.id, payload));
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

      <section className="panel tenant-governance-panel">
        <SectionHeader eyebrow="治理操作" title="租户与子账号策略" />
        <div className="workspace-grid two">
          <form className="control-form" onSubmit={handleUpdateTenant}>
            <label className="full-span">
              租户
              <select value={tenantPolicy.id} onChange={(event) => setTenantPolicy(tenantToPolicy(data?.control.tenants.find((item) => item.id === event.target.value)))}>
                {(data?.control.tenants ?? []).map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              租户名称
              <input value={tenantPolicy.name} onChange={(event) => setTenantPolicy({ ...tenantPolicy, name: event.target.value })} />
            </label>
            <label>
              状态
              <select value={tenantPolicy.status} onChange={(event) => setTenantPolicy({ ...tenantPolicy, status: event.target.value })}>
                <option value="active">启用</option>
                <option value="paused">暂停</option>
              </select>
            </label>
            <label className="full-span">
              节点访问
              <input value={tenantPolicy.nodeAccess} onChange={(event) => setTenantPolicy({ ...tenantPolicy, nodeAccess: event.target.value })} />
            </label>
            <label>
              月度 GB 配额
              <input type="number" value={tenantPolicy.monthlyTrafficGb} onChange={(event) => setTenantPolicy({ ...tenantPolicy, monthlyTrafficGb: Number(event.target.value) })} />
            </label>
            <label>
              单节点 GB 配额
              <input type="number" value={tenantPolicy.perNodeTrafficGb} onChange={(event) => setTenantPolicy({ ...tenantPolicy, perNodeTrafficGb: Number(event.target.value) })} />
            </label>
            <label>
              最大连接数
              <input type="number" value={tenantPolicy.maxConnections} onChange={(event) => setTenantPolicy({ ...tenantPolicy, maxConnections: Number(event.target.value) })} />
            </label>
            <button className="primary-button" disabled={Boolean(busy) || controlsDisabled || !tenantPolicy.id} type="submit">
              保存租户策略
            </button>
          </form>

          <form className="control-form" onSubmit={handleUpdatePanelUser}>
            <label className="full-span">
              子账号
              <select value={userPolicy.id} onChange={(event) => setUserPolicy(userToPolicy(data?.control.users.find((item) => item.id === event.target.value)))}>
                {(data?.control.users ?? []).map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.username}
                  </option>
                ))}
              </select>
            </label>
            <label>
              用户名
              <input value={userPolicy.username} onChange={(event) => setUserPolicy({ ...userPolicy, username: event.target.value })} />
            </label>
            <label>
              状态
              <select value={userPolicy.status} onChange={(event) => setUserPolicy({ ...userPolicy, status: event.target.value })}>
                <option value="active">启用</option>
                <option value="paused">暂停</option>
              </select>
            </label>
            <label>
              租户
              <select value={userPolicy.tenantId} onChange={(event) => setUserPolicy({ ...userPolicy, tenantId: event.target.value })}>
                <option value="">主租户</option>
                {(data?.control.tenants ?? []).map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              临时密码
              <input minLength={10} placeholder="留空则不变" type="password" value={userPolicy.password} onChange={(event) => setUserPolicy({ ...userPolicy, password: event.target.value })} />
            </label>
            <label className="full-span">
              节点访问
              <input value={userPolicy.nodeAccess} onChange={(event) => setUserPolicy({ ...userPolicy, nodeAccess: event.target.value })} />
            </label>
            <label>
              月度 GB 配额
              <input type="number" value={userPolicy.monthlyTrafficGb} onChange={(event) => setUserPolicy({ ...userPolicy, monthlyTrafficGb: Number(event.target.value) })} />
            </label>
            <label>
              单节点 GB 配额
              <input type="number" value={userPolicy.perNodeTrafficGb} onChange={(event) => setUserPolicy({ ...userPolicy, perNodeTrafficGb: Number(event.target.value) })} />
            </label>
            <label>
              最大连接数
              <input type="number" value={userPolicy.maxConnections} onChange={(event) => setUserPolicy({ ...userPolicy, maxConnections: Number(event.target.value) })} />
            </label>
            <button className="ghost-button" disabled={Boolean(busy) || controlsDisabled || !userPolicy.id} type="submit">
              保存子账号策略
            </button>
          </form>
        </div>
      </section>

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

function tenantToPolicy(tenant?: Tenant) {
  return {
    id: tenant?.id ?? "",
    name: tenant?.name ?? "",
    status: tenant?.status ?? "active",
    nodeAccess: (tenant?.nodeAccess ?? []).join(","),
    monthlyTrafficGb: bytesToGb(tenant?.monthlyTrafficQuota ?? 0),
    perNodeTrafficGb: bytesToGb(tenant?.perNodeTrafficQuota ?? 0),
    maxConnections: tenant?.maxConnections ?? 0
  };
}

function userToPolicy(user?: PanelUser) {
  return {
    id: user?.id ?? "",
    tenantId: user?.tenantId ?? "",
    username: user?.username ?? "",
    password: "",
    status: user?.status ?? "active",
    nodeAccess: (user?.nodeAccess ?? []).join(","),
    monthlyTrafficGb: bytesToGb(user?.monthlyTrafficQuota ?? 0),
    perNodeTrafficGb: bytesToGb(user?.perNodeTrafficQuota ?? 0),
    maxConnections: user?.maxConnections ?? 0
  };
}

function bytesToGb(value: number): number {
  if (!Number.isFinite(value) || value <= 0) {
    return 0;
  }
  return Math.round(value / 1024 / 1024 / 1024);
}

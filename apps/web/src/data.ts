export type AgentStatus = "online" | "busy" | "idle" | "offline";

export type Agent = {
  id: string;
  name: string;
  role: string;
  status: AgentStatus;
  queue: number;
  successRate: number;
  latency: string;
  cost: string;
  updatedAt: string;
};

export const agents: Agent[] = [
  {
    id: "ag-104",
    name: "Atlas Planner",
    role: "规划与拆解",
    status: "online",
    queue: 8,
    successRate: 98.2,
    latency: "1.2s",
    cost: "$18.42",
    updatedAt: "刚刚"
  },
  {
    id: "ag-219",
    name: "Vector Analyst",
    role: "数据分析",
    status: "busy",
    queue: 14,
    successRate: 95.6,
    latency: "2.4s",
    cost: "$31.08",
    updatedAt: "2 分钟前"
  },
  {
    id: "ag-331",
    name: "Beacon Runner",
    role: "自动化执行",
    status: "idle",
    queue: 2,
    successRate: 99.1,
    latency: "0.9s",
    cost: "$12.77",
    updatedAt: "8 分钟前"
  },
  {
    id: "ag-407",
    name: "Sentinel Guard",
    role: "策略审计",
    status: "offline",
    queue: 0,
    successRate: 91.8,
    latency: "--",
    cost: "$6.19",
    updatedAt: "维护中"
  }
];

export const taskRows = [
  { name: "渠道归因日报", owner: "Vector Analyst", state: "运行中", progress: 76 },
  { name: "客服质检采样", owner: "Sentinel Guard", state: "等待节点", progress: 34 },
  { name: "销售线索评分", owner: "Atlas Planner", state: "已排队", progress: 18 }
];

export const nodeRows = [
  { name: "hk-prod-a01", region: "Hong Kong", load: 68, health: "稳定" },
  { name: "sg-edge-b03", region: "Singapore", load: 52, health: "稳定" },
  { name: "jp-batch-c02", region: "Tokyo", load: 81, health: "高负载" }
];

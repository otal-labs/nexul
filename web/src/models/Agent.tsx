// The Agent page's open record, as a workspace-relative path (ADR 0211); "defaults" is every project's run settings.
export type AgentKind = "plays" | "computers" | "projects";

export const AGENT_PATH = "/agent";

export const DEFAULTS_ID = "defaults";

export const agentPath = (kind: AgentKind, id: string) => `${AGENT_PATH}/${kind}/${encodeURIComponent(id)}`;

export const isAgentKind = (value: string | undefined): value is AgentKind => value === "plays" || value === "computers" || value === "projects";

// The search param that opens the Computers dialog on any page: "" the first computer, a computer's id, "defaults", or "project:<id>".
export const COMPUTERS_PARAM = "computers";

// With a computer's id, opens that computer's Set up step beside it.
export const SETUP_PARAM = "setup";

export const PROJECT_TARGET_PREFIX = "project:";

export const projectTarget = (projectId: string) => `${PROJECT_TARGET_PREFIX}${projectId}`;

export const computersSearch = (target = "") => `?${COMPUTERS_PARAM}=${encodeURIComponent(target)}`;

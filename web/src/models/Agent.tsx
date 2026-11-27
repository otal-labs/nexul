import { PLAY_TYPES, type PlayType } from "@/models/Play";

export const AGENT_PATH = "/agent";

// One tab per play type; the tab is the path's one segment and the open row is `?open=`.
export type AgentTab = PlayType;

export const AGENT_TABS: AgentTab[] = ["ticket", "doc", "stack", "interview"];

export const AGENT_TAB_LABELS: Record<AgentTab, string> = {
  ticket: "Ticket plays",
  doc: "Doc plays",
  stack: "Stack plays",
  interview: "Interview plays",
};

export const isAgentTab = (value: string | undefined): value is AgentTab => !!value && (PLAY_TYPES as readonly string[]).includes(value);

// The search param holding the open play or auto play, and the one the Computers dialog reads.
export const OPEN_PARAM = "open";
export const COMPUTERS_PARAM = "computers";

export const agentTabPath = (tab: AgentTab) => `${AGENT_PATH}/${tab}`;

// A play's row, open: what a link to a play points at.
export const agentPlayPath = (tab: AgentTab, id: string) => `${agentTabPath(tab)}?${OPEN_PARAM}=${encodeURIComponent(id)}`;

// Where the Computers dialog opens for a computer, "defaults" or "project:<id>".
export const computersDialogPath = (target: string) => `${agentTabPath("ticket")}?${COMPUTERS_PARAM}=${encodeURIComponent(target)}`;

export const agentPath = (kind: "computers" | "projects", id: string) => `${AGENT_PATH}/${kind}/${encodeURIComponent(id)}`;

export const DEFAULTS_ID = "defaults";

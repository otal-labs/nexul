import { z } from "zod";

import type { RouteArea } from "@/models/Access";

// Mirrors internal/tenancy/model.go's Workspace wire shape (ticket 07).
export interface Workspace {
  id: string;
  name: string;
  // Names the workspace in every URL: /<slug>/board. Kept through a rename unless changed on purpose.
  slug: string;
  // A free-text template with {ticket.Field} placeholders; any member reads it, only workspaces:write can change it.
  mention_chip_template: string;
  created_at: string;
  updated_at: string;
}

// Matches the column default on workspaces so a chip looks the same before the workspace list resolves.
export const DEFAULT_MENTION_CHIP_TEMPLATE = "{ticket.Ticket} {ticket.Status}";

export const SaveWorkspaceFormSchema = z.object({
  name: z.string().trim().min(1, "Workspace name is required"),
});

export type SaveWorkspaceFormData = z.infer<typeof SaveWorkspaceFormSchema>;

// Same rule as internal/tenancy's Slugify, so a name previews the slug the server derives from it.
export const slugify = (name: string): string =>
  name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 48)
    .replace(/-+$/, "") || "workspace";

// Prefixes a workspace-relative path ("/board", "/tickets/WEB-1") with the workspace's slug.
export const workspacePath = (slug: string, path: string): string => `/${slug}${path === "/" ? "" : path}`;

// What the viewer may open in the workspace being switched to.
export interface WorkspaceAccess {
  canOpen: (area: RouteArea) => boolean;
  // Its Configuration sections the viewer may open, in nav order.
  configurationSections: readonly string[];
}

// Section → the area a switch needs, and where it lands: items are workspace-bound, so only the section survives.
const sectionLandings: Record<string, { area?: RouteArea; path: string }> = {
  board: { area: "tickets", path: "/board" },
  tickets: { area: "tickets", path: "/board" },
  projects: { area: "tickets", path: "/board" },
  docs: { area: "docs", path: "/docs" },
  memories: { area: "memories", path: "/memories" },
  chat: { path: "/chat" },
  inbox: { path: "/inbox" },
  runners: { area: "runners", path: "/runners" },
  topology: { area: "topology", path: "/topology" },
  stacks: { area: "topology", path: "/topology" },
  automations: { area: "automations", path: "/automations" },
  wizard: { area: "newProject", path: "/wizard/project/project" },
};

const configurationLanding = (section: string | undefined, access: WorkspaceAccess): string | undefined => {
  if (section && access.configurationSections.includes(section)) return `/configuration/${section}`;
  const first = access.configurationSections.find((s) => s !== "danger");
  return first && `/configuration/${first}`;
};

// Where the workspace switcher lands: the same section in the target workspace, never the item (a project, doc,
// ticket, or conversation belongs to the workspace it was opened in); home when the viewer can't open it there.
export const switchWorkspacePath = (pathname: string, targetSlug: string, access: WorkspaceAccess): string => {
  const [section, next] = pathname.split("/").filter(Boolean).slice(1);
  const home = workspacePath(targetSlug, "/");
  if (section === "configuration") {
    const landing = configurationLanding(next, access);
    return landing ? workspacePath(targetSlug, landing) : home;
  }
  const landing = section ? sectionLandings[section] : undefined;
  if (!landing || (landing.area && !access.canOpen(landing.area))) return home;
  return workspacePath(targetSlug, landing.path);
};

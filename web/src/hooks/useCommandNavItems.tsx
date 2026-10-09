import {
  BrainIcon,
  ClipboardListIcon,
  CpuIcon,
  FileTextIcon,
  InboxIcon,
  LayoutDashboardIcon,
  MessagesSquareIcon,
  NetworkIcon,
  SettingsIcon,
  SlidersHorizontalIcon,
  UserCogIcon,
  WorkflowIcon,
} from "lucide-react";
import { useNavigate } from "react-router";

import { useAreaAccess, useCanOpen } from "@/hooks/AccessHooks";
import { useSidebarProject } from "@/hooks/useSidebarProject";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useFetchMyRole } from "@/hooks/WorkspaceHooks";
import { useWorkspaceStore } from "@/stores/workspaceStore";
import type { CommandGroup, CommandItem } from "@/models/Command";
import { hasPermission, projectPermissions } from "@/models/Permission";
import { boardPath, interviewPath, projectSettingsPath, projectToken } from "@/models/Project";

// Every page the sidebar reaches, under the same permissions, and every project's board.
export const useCommandNavItems = (): CommandGroup[] => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const { projects, current } = useSidebarProject();
  const workspaceId = useWorkspaceStore((s) => s.selectedWorkspaceId);
  const { data: role } = useFetchMyRole(workspaceId);
  const can = useAreaAccess(current?.id);
  const canOpen = useCanOpen();
  const go = (path: string) => () => void navigate(wsPath(path));

  const page = (id: string, label: string, icon: CommandItem["icon"], path: string, shown = true): CommandItem[] =>
    shown ? [{ id: `page:${id}`, label, icon, run: go(path), keywords: "go to page" }] : [];

  const token = current ? projectToken(current) : "";
  const pages: CommandItem[] = [
    ...page("inbox", "Inbox", InboxIcon, "/inbox"),
    ...page("chat", "Chat", MessagesSquareIcon, "/chat"),
    ...page("board", "Board", LayoutDashboardIcon, current ? boardPath(current) : "/board", !!current && !!can?.("tickets")),
    ...page("interview", "Interview", ClipboardListIcon, interviewPath(token), !!current && !!can?.("memories")),
    ...page("docs", "Docs", FileTextIcon, "/docs", !!current && !!can?.("docs")),
    ...page("memories", "Memories", BrainIcon, "/memories", !!current && !!can?.("memories")),
    ...page("project-settings", "Project settings", SettingsIcon, projectSettingsPath(token), !!current && !!can?.("projects")),
    ...page("runners", "Runners", CpuIcon, "/runners", canOpen("runners") === true),
    ...page("topology", "Topology", NetworkIcon, "/topology", canOpen("topology") === true),
    ...page("automations", "Automations", WorkflowIcon, "/automations", canOpen("automations") === true),
    ...page("configuration", "Configuration", SlidersHorizontalIcon, "/configuration", canOpen("configuration") === true),
    { id: "page:settings", label: "Your settings", icon: UserCogIcon, keywords: "go to page profile account", run: () => void navigate("/settings") },
  ];

  const boards: CommandItem[] = (projects ?? [])
    .filter((project) => hasPermission(projectPermissions(role, project.id), "tickets:read"))
    .map((project) => ({
      id: `board:${project.id}`,
      label: project.name,
      icon: LayoutDashboardIcon,
      hint: project.prefix,
      keywords: "board project",
      run: go(boardPath(project)),
    }));

  return [
    { heading: "Go to", items: pages },
    { heading: "Boards", items: boards },
  ];
};

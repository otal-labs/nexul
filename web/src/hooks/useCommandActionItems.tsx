import { ArrowRightLeftIcon, FilePlusIcon, HashIcon, MessageSquarePlusIcon, MoonIcon, PaletteIcon, SquarePlusIcon, SunIcon } from "lucide-react";
import { useNavigate } from "react-router";
import { useShallow } from "zustand/react/shallow";

import { useAreaAccess } from "@/hooks/AccessHooks";
import { useCreateDocDialog } from "@/hooks/useCreateDocDialog";
import { useCreateTicketDialog } from "@/hooks/useCreateTicketDialog";
import { useNewConversationDialogs } from "@/hooks/useNewConversationDialogs";
import { useSidebarProject } from "@/hooks/useSidebarProject";
import { useSwitchWorkspace } from "@/hooks/useSwitchWorkspace";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { useThemeStore } from "@/stores/themeStore";
import { ThemeName } from "@/enums/Theme";
import type { CommandGroup, CommandItem } from "@/models/Command";
import { docPath, projectToken } from "@/models/Project";
import { THEME_DEFINITIONS } from "@/lib/themePalettes";

// The create dialogs that already exist, the other workspaces, and the theme. Palettes only show once something is typed.
export const useCommandActionItems = (browsing: boolean): CommandGroup[] => {
  const navigate = useNavigate();
  const wsPath = useWorkspacePath();
  const { current: project } = useSidebarProject();
  const { workspaces, current: workspace, switchTo } = useSwitchWorkspace();
  const can = useAreaAccess();
  const createTicket = useCreateTicketDialog(project?.id ?? "");
  const createDoc = useCreateDocDialog(project?.id ?? "");
  const { openNewChannel, openNewDM } = useNewConversationDialogs(workspace?.id ?? "", (conversation) => {
    if (conversation.kind === "voice_channel") return;
    void navigate(wsPath(`/chat/${conversation.id}`));
  });
  const { theme, themeId, setTheme, setThemeId } = useThemeStore(
    useShallow((s) => ({ theme: s.theme, themeId: s.themeId, setTheme: s.setTheme, setThemeId: s.setThemeId })),
  );

  const create: CommandItem[] = [];
  if (project && createTicket) {
    const run = async () => {
      const id = await createTicket();
      if (id) void navigate(wsPath(`/tickets/${id}`));
    };
    create.push({ id: "create:ticket", label: "New ticket", icon: SquarePlusIcon, hint: project.prefix, keywords: "create", run: () => void run() });
  }
  if (project && createDoc) {
    const run = async () => {
      const id = await createDoc();
      if (id) void navigate(wsPath(docPath(projectToken(project), id)));
    };
    create.push({ id: "create:doc", label: "New doc", icon: FilePlusIcon, hint: project.prefix, keywords: "create document", run: () => void run() });
  }
  if (can?.("editChannels")) create.push({ id: "create:channel", label: "New channel", icon: HashIcon, keywords: "create chat", run: () => void openNewChannel(false) });
  if (can?.("newConversation")) {
    create.push({ id: "create:dm", label: "New direct message", icon: MessageSquarePlusIcon, keywords: "create chat dm", run: () => void openNewDM() });
  }

  const others: CommandItem[] = (workspaces ?? [])
    .filter((candidate) => candidate.id !== workspace?.id)
    .map((candidate) => ({
      id: `workspace:${candidate.id}`,
      label: `Switch to ${candidate.name}`,
      icon: ArrowRightLeftIcon,
      keywords: "workspace",
      run: () => void switchTo(candidate.id),
    }));

  const dark = theme === ThemeName.Dark;
  const mode: CommandItem = {
    id: "theme:mode",
    label: dark ? "Switch to light mode" : "Switch to dark mode",
    icon: dark ? SunIcon : MoonIcon,
    keywords: "theme appearance",
    run: () => setTheme(dark ? ThemeName.Light : ThemeName.Dark),
  };
  const palettes: CommandItem[] = THEME_DEFINITIONS.map((definition) => ({
    id: `theme:${definition.id}`,
    label: `${definition.label} theme`,
    icon: PaletteIcon,
    ...(definition.id === themeId && { hint: "current" }),
    keywords: "palette colour color appearance",
    run: () => setThemeId(definition.id),
  }));

  return [
    { heading: "Create", items: create },
    { heading: "Workspaces", items: others },
    { heading: "Theme", items: browsing ? [mode] : [mode, ...palettes] },
  ];
};

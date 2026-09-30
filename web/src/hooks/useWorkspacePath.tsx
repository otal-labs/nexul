import { useCallback } from "react";
import { useLocation } from "react-router";

import { useWorkspaceStore } from "@/stores/workspaceStore";
import { workspacePath } from "@/models/Workspace";

// Builds links into the selected workspace: wsPath("/board") is "/<slug>/board".
export const useWorkspacePath = (): ((path: string) => string) => {
  const slug = useWorkspaceStore((s) => s.selectedWorkspaceSlug);
  return useCallback((path: string) => workspacePath(slug, path), [slug]);
};

// The current path inside the selected workspace ("/board/WEB" on "/<slug>/board/WEB"); unprefixed pages pass through.
export const useWorkspacePathname = (): string => {
  const { pathname } = useLocation();
  const prefix = `/${useWorkspaceStore((s) => s.selectedWorkspaceSlug)}`;
  if (prefix === "/" || (pathname !== prefix && !pathname.startsWith(`${prefix}/`))) return pathname;
  return pathname.slice(prefix.length) || "/";
};

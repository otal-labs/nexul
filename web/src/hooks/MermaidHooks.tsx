import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { useThemeStore } from "@/stores/themeStore";
import { ThemeName } from "@/enums/Theme";
import { renderMermaid } from "@/lib/mermaid";
import { pause } from "@/lib/pause";

export const getMermaidSvgKey = "getMermaidSvg";

const typingPauseMs = 300;

// Redraws once typing pauses, keeping the last drawing on screen meanwhile; a diagram that doesn't parse is never retried.
export const useMermaidSvg = (code: string) => {
  const dark = useThemeStore((s) => s.theme === ThemeName.Dark);
  return useQuery({
    queryKey: [getMermaidSvgKey, code, dark],
    staleTime: Infinity,
    retry: false,
    placeholderData: keepPreviousData,
    queryFn: async ({ signal }) => {
      await pause(typingPauseMs, signal);
      return renderMermaid(code, dark);
    },
  });
};

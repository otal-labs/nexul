import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useShallow } from "zustand/react/shallow";

import { htmlPageDocument } from "@nexul/client-core/htmlPage";
import { readHtmlPageTheme } from "@/lib/htmlPageTheme";
import { pause } from "@/lib/pause";
import { useThemeStore } from "@/stores/themeStore";

export const getHtmlPageDocumentKey = "getHtmlPageDocument";

const typingPauseMs = 300;

// Rebuilds the page once typing pauses, keeping the last one on screen meanwhile; a theme change rebuilds it in the new colours.
export const useHtmlPageDocument = (code: string) => {
  const { theme, themeId } = useThemeStore(useShallow((s) => ({ theme: s.theme, themeId: s.themeId })));
  return useQuery({
    queryKey: [getHtmlPageDocumentKey, code, theme, themeId],
    staleTime: Infinity,
    retry: false,
    placeholderData: keepPreviousData,
    queryFn: async ({ signal }) => {
      await pause(typingPauseMs, signal);
      return { html: htmlPageDocument(code, readHtmlPageTheme(theme)), appearance: theme };
    },
  });
};

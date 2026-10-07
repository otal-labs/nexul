import { router, type Href } from "expo-router";
import { useMemo } from "react";
import { Linking } from "react-native";
import { EnrichedMarkdownText, type MarkdownStyle } from "react-native-enriched-markdown";
import { useCSSVariable } from "uniwind";

import { useAreaAccess } from "@/hooks/WorkspaceHooks";
import { useSessionStore } from "@/stores/sessionStore";

const sans = "Inter";
const mono = "JetBrains Mono";

// The native renderer cannot read the stylesheet, so it takes the same tokens as values.
const useMarkdownStyle = (): MarkdownStyle => {
  const [fg, muted, line, fill] = useCSSVariable([
    "--color-foreground",
    "--color-muted-foreground",
    "--color-border",
    "--color-surface-2",
  ]);
  const [foreground, mutedForeground, border, surface] = [String(fg), String(muted), String(line), String(fill)];
  return useMemo(() => {
    const body = { fontFamily: sans, fontSize: 15, lineHeight: 21, color: foreground };
    const heading = { fontFamily: sans, fontWeight: "600", color: foreground };
    return {
      paragraph: body,
      h1: { ...heading, fontSize: 20 },
      h2: { ...heading, fontSize: 18 },
      h3: { ...heading, fontSize: 16 },
      strong: { color: foreground },
      em: { color: foreground },
      link: { color: foreground, underline: true },
      list: { ...body, bulletColor: mutedForeground, markerColor: mutedForeground },
      blockquote: { ...body, color: mutedForeground, borderColor: border, backgroundColor: "transparent" },
      code: { fontFamily: mono, color: foreground, backgroundColor: surface, borderColor: border },
      thematicBreak: { color: border, height: 1 },
      codeBlock: {
        fontFamily: mono,
        fontSize: 13,
        color: foreground,
        backgroundColor: surface,
        borderColor: border,
        borderWidth: 1,
        borderRadius: 6,
        padding: 10,
      },
    };
  }, [foreground, mutedForeground, border, surface]);
};

// Web doc and ticket pages under a workspace slug, which a ticket key needs, and a mention's own /docs/<id> or /tickets/<id>.
// A ticket is only in-app for a viewer who has the Board tab; for anyone else it is the web page, which says it isn't found.
const inAppRoute = (path: string, canReadTickets: boolean): Href | null => {
  const doc = /^(?:\/[a-z0-9-]+\/docs\/(?:[^/?#]+\/)?|\/docs\/)([^/?#]+)(?=[?#]|$)/.exec(path);
  if (doc?.[1]) return `/more/docs/${doc[1]}`;
  const ticket = /^(?:\/([a-z0-9-]+))?\/tickets\/([^/?#]+)(?=[?#]|$)/.exec(path);
  if (!ticket?.[2] || !canReadTickets) return null;
  if (ticket[1]) return { pathname: "/board/ticket/[id]", params: { id: ticket[2], workspace: ticket[1] } };
  return `/board/ticket/${ticket[2]}`;
};

export const openLink = (url: string, canReadTickets: boolean) => {
  const host = useSessionStore.getState().host ?? "";
  const path = host && url.startsWith(`${host}/`) ? url.slice(host.length) : url;
  const route = inAppRoute(path, canReadTickets);
  if (route) {
    router.push(route, { withAnchor: true });
    return;
  }
  // Any other relative link is a page of the web app, which only the browser can show.
  void Linking.openURL(path.startsWith("/") ? `${host}${path}` : url);
};

interface MessageMarkdownProps {
  markdown: string;
}

export const MessageMarkdown = ({ markdown }: MessageMarkdownProps) => {
  const canReadTickets = useAreaAccess()?.("tickets") ?? false;
  return (
    <EnrichedMarkdownText
      markdown={markdown}
      flavor="github"
      markdownStyle={useMarkdownStyle()}
      onLinkPress={({ url }) => openLink(url, canReadTickets)}
    />
  );
};

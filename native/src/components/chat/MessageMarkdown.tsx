import { router, type Href } from "expo-router";
import { useMemo } from "react";
import { Linking } from "react-native";
import { EnrichedMarkdownText, type MarkdownStyle } from "react-native-enriched-markdown";
import { useCSSVariable } from "uniwind";

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

// Both doc URL shapes the web serves, /docs/<id> and /docs/<projectToken>/<id>, and ticket URLs, relative or on the instance.
const inAppRoute = (path: string): Href | null => {
  const doc = /^\/docs\/(?:[^/?#]+\/)?([^/?#]+)(?=[?#]|$)/.exec(path);
  if (doc?.[1]) return `/more/docs/${doc[1]}`;
  const ticket = /^\/tickets\/([^/?#]+)(?=[?#]|$)/.exec(path);
  if (ticket?.[1]) return `/board/ticket/${ticket[1]}`;
  return null;
};

const openLink = (url: string) => {
  const host = useSessionStore.getState().host ?? "";
  const path = host && url.startsWith(`${host}/`) ? url.slice(host.length) : url;
  const route = inAppRoute(path);
  if (route) {
    router.push(route);
    return;
  }
  // Any other relative link is a page of the web app, which only the browser can show.
  void Linking.openURL(path.startsWith("/") ? `${host}${path}` : url);
};

interface MessageMarkdownProps {
  markdown: string;
}

export const MessageMarkdown = ({ markdown }: MessageMarkdownProps) => (
  <EnrichedMarkdownText
    markdown={markdown}
    flavor="github"
    markdownStyle={useMarkdownStyle()}
    onLinkPress={({ url }) => openLink(url)}
  />
);

import { router, type Href } from "expo-router";
import { useMemo, useState } from "react";
import { Linking } from "react-native";
import { EnrichedMarkdownText, type MarkdownStyle } from "react-native-enriched-markdown";
import { useCSSVariable } from "uniwind";

import { formatDiscordTime, hasRelativeTime, splitDiscordTimes } from "@nexul/client-core/discordTime";

import { DiscordTimePopover, type OpenTime } from "@/components/chat/DiscordTimePopover";
import { useAreaAccess } from "@/hooks/WorkspaceHooks";
import { useClockStore } from "@/stores/clockStore";
import { useSessionStore } from "@/stores/sessionStore";

const sans = "Inter";
const mono = "JetBrains Mono";

// The native renderer cannot read the stylesheet, so it takes the same tokens as values; on your own bubble the ink is the
// brand's foreground and code sits on a darker tint of the bubble.
const useMarkdownStyle = (own: boolean): MarkdownStyle => {
  const [fg, muted, line, fill, brandFg] = useCSSVariable([
    "--color-foreground",
    "--color-muted-foreground",
    "--color-border",
    "--color-surface-2",
    "--color-brand-foreground",
  ]);
  const ink = own ? String(brandFg) : String(fg);
  const [foreground, mutedForeground, border, surface] = own
    ? [ink, ink, "#00000033", "#0000001f"]
    : [ink, String(muted), String(line), String(fill)];
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
      inlineImage: { size: 15 },
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

const TIME_LINK = "nexul-time:";

// The renderer draws the markdown natively, so each <t:…> goes in as a link carrying its code, which a tap reports back.
const withTimeLinks = (markdown: string, now: number): string =>
  splitDiscordTimes(markdown)
    .map((part) => {
      if (part.kind === "text") return part.text;
      const label = formatDiscordTime(part.time, now).replace(/[[\]\\]/g, "\\$&");
      return `[${label}](${TIME_LINK}${encodeURIComponent(part.time.code)})`;
    })
    .join("");

const decoded = (s: string): string => {
  try {
    return decodeURIComponent(s);
  } catch {
    return s;
  }
};

const timeAt = (url: string): OpenTime | null => {
  if (!url.startsWith(TIME_LINK)) return null;
  const part = splitDiscordTimes(decoded(url.slice(TIME_LINK.length)))[0];
  return part?.kind === "time" ? { time: part.time, at: null } : null;
};

interface MessageMarkdownProps {
  markdown: string;
  own?: boolean;
}

export const MessageMarkdown = ({ markdown, own = false }: MessageMarkdownProps) => {
  const canReadTickets = useAreaAccess()?.("tickets") ?? false;
  const live = useMemo(() => hasRelativeTime(markdown), [markdown]);
  // Only a message with a relative time follows the clock, so the minute tick leaves every other one alone.
  const now = useClockStore((s) => (live ? s.now : 0));
  const timed = useMemo(() => withTimeLinks(markdown, now), [markdown, now]);
  const [open, setOpen] = useState<OpenTime | null>(null);
  const onLinkPress = ({ url }: { url: string }) => {
    const time = timeAt(url);
    if (time) return setOpen(time);
    openLink(url, canReadTickets);
  };
  return (
    <>
      <EnrichedMarkdownText markdown={timed} flavor="github" markdownStyle={useMarkdownStyle(own)} onLinkPress={onLinkPress} />
      <DiscordTimePopover open={open} onClose={() => setOpen(null)} />
    </>
  );
};

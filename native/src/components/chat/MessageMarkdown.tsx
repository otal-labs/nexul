import { useMemo } from "react";
import { Linking } from "react-native";
import { EnrichedMarkdownText, type MarkdownStyle } from "react-native-enriched-markdown";
import { useCSSVariable } from "uniwind";

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

interface MessageMarkdownProps {
  markdown: string;
  /** Overrides the default "open in the system browser"; a caller with internal links routes them here. */
  onLinkPress?: (url: string) => void;
}

export const MessageMarkdown = ({ markdown, onLinkPress }: MessageMarkdownProps) => (
  <EnrichedMarkdownText
    markdown={markdown}
    flavor="github"
    markdownStyle={useMarkdownStyle()}
    onLinkPress={({ url }) => (onLinkPress ? onLinkPress(url) : void Linking.openURL(url))}
  />
);

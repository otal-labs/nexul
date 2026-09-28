import { useRouter } from "expo-router";
import { Linking } from "react-native";

import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import type { RichMark, RichNode } from "@/models/Doc";

interface DocInlineProps {
  nodes: RichNode[];
}

// Mirrors richtext.MentionHref: a link to another doc or ticket pushes onto this app's own stack instead of
// leaving it; anything else opens in the system browser.
const openLink = (router: ReturnType<typeof useRouter>, href: string) => {
  const docMatch = /^\/docs\/([^/]+)$/.exec(href);
  if (docMatch?.[1]) {
    router.push(`/more/docs/${docMatch[1]}`);
    return;
  }
  const ticketMatch = /^\/tickets\/([^/]+)$/.exec(href);
  if (ticketMatch?.[1]) {
    router.push(`/board/ticket/${ticketMatch[1]}`);
    return;
  }
  void Linking.openURL(href);
};

const markClassName = (marks: RichMark[] | undefined): string =>
  cn(
    marks?.some((m) => m.type === "bold") && "font-semibold",
    marks?.some((m) => m.type === "italic") && "italic",
    marks?.some((m) => m.type === "strike") && "line-through",
    marks?.some((m) => m.type === "code") && "rounded bg-muted px-1 font-mono",
  );

// Renders one run of inline nodes as sibling Text spans; the caller supplies the wrapping <Text>.
export const DocInline = ({ nodes }: DocInlineProps) => {
  const router = useRouter();
  return (
    <>
      {nodes.map((node, index) => {
        if (node.type === "hardBreak") return <Text key={index}>{"\n"}</Text>;
        if (node.type === "mention") {
          return (
            <Text key={index} className="font-mono">
              {String(node.attrs?.label ?? "")}
            </Text>
          );
        }
        if (node.type !== "text") return null;
        const href = node.marks?.find((m) => m.type === "link")?.attrs?.href;
        if (typeof href === "string") {
          return (
            <Text key={index} className="text-primary underline" onPress={() => openLink(router, href)}>
              {node.text ?? ""}
            </Text>
          );
        }
        return (
          <Text key={index} className={markClassName(node.marks)}>
            {node.text ?? ""}
          </Text>
        );
      })}
    </>
  );
};

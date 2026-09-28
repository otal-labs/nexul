import { Image, ScrollView, View } from "react-native";

import { DocInline } from "@/components/docs/DocInline";
import { Text } from "@/components/ui/text";
import { cn } from "@/lib/utils";
import type { RichNode } from "@/models/Doc";
import { readSessionToken, useSessionStore } from "@/stores/sessionStore";

interface DocBlockNodeProps {
  node: RichNode;
  depth?: number;
}

const HEADING_CLASS: Record<number, string> = {
  1: "text-xl font-semibold",
  2: "text-lg font-semibold",
  3: "text-base font-semibold",
};

const DEFAULT_HEADING_CLASS = "text-base font-semibold";

const headingClassName = (level: unknown): string => HEADING_CLASS[Number(level)] ?? DEFAULT_HEADING_CLASS;

const codeText = (node: RichNode): string => (node.content ?? []).map((child) => child.text ?? "").join("");

const ATTACHMENT_PREFIX = "/api/attachments/";

// Attachment images need the bearer token; RN's Image passes headers straight to the network request,
// so no blob-URL indirection (the web workaround for a bare <img>) is needed here.
const DocImage = ({ node }: { node: RichNode }) => {
  const host = useSessionStore((s) => s.host);
  const src = String(node.attrs?.src ?? "");
  const alt = String(node.attrs?.alt ?? "Image");
  if (!src) return null;
  const isAttachment = src.startsWith(ATTACHMENT_PREFIX);
  const token = isAttachment ? readSessionToken() : null;
  return (
    <Image
      accessibilityLabel={alt}
      resizeMode="contain"
      className="rounded-md bg-muted"
      // ponytail: fixed 200px height until intrinsic sizing (Image.getSize + state) earns its complexity.
      style={{ width: "100%", height: 200 }}
      source={{
        uri: isAttachment ? `${host ?? ""}${src}` : src,
        ...(token && { headers: { Authorization: `Bearer ${token}` } }),
      }}
    />
  );
};

// One node of the Tiptap tree (ADR 0026), mirroring internal/docs/richtext's renderBlock 1:1 so nothing
// the editor can produce is silently dropped.
export const DocBlockNode = ({ node, depth = 0 }: DocBlockNodeProps) => {
  if (node.type === "paragraph") {
    return (
      <Text className="leading-6">
        <DocInline nodes={node.content ?? []} />
      </Text>
    );
  }
  if (node.type === "heading") {
    return (
      <Text className={cn("leading-6", headingClassName(node.attrs?.level))}>
        <DocInline nodes={node.content ?? []} />
      </Text>
    );
  }
  if (node.type === "codeBlock") {
    return (
      <ScrollView horizontal className="rounded-md bg-muted" contentContainerClassName="p-3">
        <Text className="font-mono text-sm">{codeText(node)}</Text>
      </ScrollView>
    );
  }
  if (node.type === "image") return <DocImage node={node} />;
  if (node.type === "horizontalRule") return <View className="h-px bg-border" />;
  if (node.type === "blockquote") {
    return (
      <View className="gap-2 border-l-2 border-border pl-3">
        {(node.content ?? []).map((child, i) => (
          <DocBlockNode key={i} node={child} depth={depth} />
        ))}
      </View>
    );
  }
  if (node.type === "bulletList" || node.type === "orderedList") {
    const ordered = node.type === "orderedList";
    const start = ordered && typeof node.attrs?.start === "number" ? (node.attrs.start as number) : 1;
    return (
      <View className="gap-2">
        {(node.content ?? []).map((item, i) => {
          const [first, ...rest] = item.content ?? [];
          return (
            <View key={i} className="flex-row gap-2" style={{ paddingLeft: depth * 16 }}>
              <Text className="font-mono text-muted-foreground">{ordered ? `${start + i}.` : "•"}</Text>
              <View className="flex-1 gap-2">
                {first && <DocBlockNode node={first} depth={depth + 1} />}
                {rest.map((child, j) => (
                  <DocBlockNode key={j} node={child} depth={depth + 1} />
                ))}
              </View>
            </View>
          );
        })}
      </View>
    );
  }
  // Unrecognized node: render its inline text rather than dropping it silently (mirrors renderUnknownBlock).
  return node.content ? <DocInline nodes={node.content} /> : null;
};

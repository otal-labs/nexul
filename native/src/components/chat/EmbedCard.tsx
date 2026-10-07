import { useState } from "react";
import { Image, Pressable, View } from "react-native";

import { DiscordMarkdown } from "@/components/chat/DiscordMarkdown";
import { EmbedFieldGrid } from "@/components/chat/EmbedFieldGrid";
import { FoldBar } from "@/components/chat/FoldBar";
import { openLink } from "@/components/chat/MessageMarkdown";
import { Text } from "@/components/ui/text";
import { useAreaAccess } from "@/hooks/WorkspaceHooks";
import { useClockStore } from "@/stores/clockStore";
import { EMBED_FIELD_LIMIT, embedFold, embedFoldLabel, embedTimestamp, httpUrl, type Embed } from "@/models/Embed";
import { formatCalendarTime } from "@/lib/time";
import { cn } from "@/lib/utils";

// A clamped description shows six of its 20pt lines.
const CLAMPED_DESCRIPTION = "max-h-[120px] overflow-hidden";

// A picture that fails to load drops out rather than leaving a broken frame.
const EmbedImage = ({ uri }: { uri: string }) => {
  const [failed, setFailed] = useState(false);
  const [aspectRatio, setAspectRatio] = useState(16 / 9);
  if (failed) return null;
  return (
    <Image
      accessibilityLabel="Embed image"
      source={{ uri }}
      resizeMode="cover"
      onError={() => setFailed(true)}
      onLoad={({ nativeEvent: { source } }) => source.height > 0 && setAspectRatio(source.width / source.height)}
      style={{ aspectRatio }}
      className="mt-1 max-h-72 w-full rounded-md border border-border"
    />
  );
};

const EmbedIcon = ({ uri }: { uri: string }) => <Image source={{ uri }} className="size-4 rounded-full" />;

// The author line over the title, which links out when the sender gave it a URL.
const EmbedHeading = ({ embed: { author, title, url } }: { embed: Embed }) => {
  const href = httpUrl(url);
  const icon = httpUrl(author?.icon_url);
  const authorUrl = httpUrl(author?.url);
  const canReadTickets = useAreaAccess()?.("tickets") ?? false;
  return (
    <>
      {author && (
        <Pressable disabled={!authorUrl} onPress={() => authorUrl && openLink(authorUrl, canReadTickets)} className="flex-row items-center gap-1.5">
          {icon && <EmbedIcon uri={icon} />}
          <Text numberOfLines={1} className="shrink text-xs font-medium text-foreground/85">
            {author.name}
          </Text>
        </Pressable>
      )}
      {title && href && (
        <Text role="link" onPress={() => openLink(href, canReadTickets)} className="text-sm font-semibold underline">
          {title}
        </Text>
      )}
      {title && !href && <Text className="text-sm font-semibold">{title}</Text>}
    </>
  );
};

const EmbedFooter = ({ embed: { footer, timestamp } }: { embed: Embed }) => {
  const now = useClockStore((s) => s.now);
  const icon = httpUrl(footer?.icon_url);
  return (
    <View className="flex-row items-center gap-1.5 pt-0.5">
      {icon && <EmbedIcon uri={icon} />}
      {footer && (
        <Text numberOfLines={1} className="shrink text-[11px] font-medium text-muted-foreground">
          {footer.text}
        </Text>
      )}
      {footer && timestamp && <Text className="text-[11px] text-muted-foreground">•</Text>}
      {timestamp && <Text className="text-[11px] font-medium text-muted-foreground">{formatCalendarTime(embedTimestamp(timestamp), now)}</Text>}
    </View>
  );
};

// One embed behind a neutral rule (a sender's color would misread as status), its thumbnail above the text.
export const EmbedCard = ({ embed }: { embed: Embed }) => {
  const [open, setOpen] = useState(false);
  const fold = embedFold(embed);
  const foldLabel = embedFoldLabel(fold);
  const fields = open ? (embed.fields ?? []) : (embed.fields ?? []).slice(0, EMBED_FIELD_LIMIT);
  const image = httpUrl(embed.image?.url);
  const thumbnail = httpUrl(embed.thumbnail?.url);
  return (
    <View className="gap-1.5 border-l-2 border-muted-foreground/30 py-0.5 pl-3">
      {thumbnail && <Image accessibilityLabel="Embed thumbnail" source={{ uri: thumbnail }} className="size-16 rounded-md border border-border" />}
      <EmbedHeading embed={embed} />
      {embed.description && (
        <View className={cn(fold.longDescription && !open && CLAMPED_DESCRIPTION)}>
          <DiscordMarkdown text={embed.description} textClassName="text-sm leading-5 text-foreground/80" />
        </View>
      )}
      {fields.length > 0 && <EmbedFieldGrid fields={fields} />}
      {foldLabel && <FoldBar label={open ? "Show less" : foldLabel} open={open} onToggle={() => setOpen(!open)} />}
      {image && <EmbedImage uri={image} />}
      {(embed.footer || embed.timestamp) && <EmbedFooter embed={embed} />}
    </View>
  );
};

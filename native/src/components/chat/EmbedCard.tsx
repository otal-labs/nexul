import { useState } from "react";
import { Image, Pressable, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { DiscordMarkdown } from "@/components/chat/DiscordMarkdown";
import { EmbedFieldGrid } from "@/components/chat/EmbedFieldGrid";
import { toneBg, toneIcon, toneVar } from "@/components/chat/EmbedTone";
import { FoldBar } from "@/components/chat/FoldBar";
import { openLink } from "@/components/chat/MessageMarkdown";
import { Text } from "@/components/ui/text";
import { useBotMediaSource } from "@/hooks/BotMediaHooks";
import { useAreaAccess } from "@/hooks/WorkspaceHooks";
import { useClockStore } from "@/stores/clockStore";
import { EMBED_FIELD_LIMIT, embedCardTone, embedFold, embedFoldLabel, embedTimestamp, httpUrl, type Embed, type EmbedTone } from "@/models/Embed";
import { formatCalendarTime } from "@/lib/time";
import { cn } from "@/lib/utils";

// A clamped description shows six of its 20pt lines; the fold opens the rest.
const CLAMPED_DESCRIPTION = "max-h-[120px] overflow-hidden";

// A picture that fails to load drops out rather than leaving a broken frame.
const EmbedImage = ({ uri }: { uri: string }) => {
  const source = useBotMediaSource(uri);
  const [failed, setFailed] = useState(false);
  const [aspectRatio, setAspectRatio] = useState(16 / 9);
  if (failed || !source) return null;
  return (
    <Image
      accessibilityLabel="Embed image"
      source={source}
      resizeMode="cover"
      onError={() => setFailed(true)}
      onLoad={({ nativeEvent: { source } }) => source.height > 0 && setAspectRatio(source.width / source.height)}
      style={{ aspectRatio }}
      className="mt-1 max-h-72 w-full rounded-md border border-border"
    />
  );
};

const EmbedIcon = ({ uri }: { uri: string }) => {
  const source = useBotMediaSource(uri);
  return source && <Image source={source} className="size-4 rounded-full" />;
};

const EmbedThumbnail = ({ uri }: { uri: string }) => {
  const source = useBotMediaSource(uri);
  return source && <Image accessibilityLabel="Embed thumbnail" source={source} className="size-16 rounded-md border border-border" />;
};

// The author line over the title, which leads with the post's state and links out when the sender gave it a URL.
const EmbedHeading = ({ embed: { author, title, url }, tone }: { embed: Embed; tone: EmbedTone | null }) => {
  const href = httpUrl(url);
  const icon = httpUrl(author?.icon_url);
  const authorUrl = httpUrl(author?.url);
  const canReadTickets = useAreaAccess()?.("tickets") ?? false;
  const [toneColor] = useCSSVariable([tone ? toneVar[tone] : "--color-muted-foreground"]);
  const Icon = tone && toneIcon[tone];
  return (
    <View className="gap-1">
      {author && (
        <Pressable disabled={!authorUrl} onPress={() => authorUrl && openLink(authorUrl, canReadTickets)} className="flex-row items-center gap-1.5">
          {icon && <EmbedIcon uri={icon} />}
          <Text numberOfLines={1} className="shrink text-xs font-medium text-muted-foreground">
            {author.name}
          </Text>
        </Pressable>
      )}
      {title && (
        <View className="flex-row items-start gap-2">
          {Icon && <Icon size={16} color={String(toneColor)} style={{ marginTop: 2 }} />}
          {href && (
            <Text role="link" onPress={() => openLink(href, canReadTickets)} className="min-w-0 flex-1 text-sm font-semibold leading-5 underline">
              {title}
            </Text>
          )}
          {!href && <Text className="min-w-0 flex-1 text-sm font-semibold leading-5">{title}</Text>}
        </View>
      )}
    </View>
  );
};

const EmbedFooter = ({ embed: { footer, timestamp } }: { embed: Embed }) => {
  const now = useClockStore((s) => s.now);
  const icon = httpUrl(footer?.icon_url);
  return (
    <View className="min-w-0 flex-1 flex-row items-center gap-1.5">
      {icon && <EmbedIcon uri={icon} />}
      {footer && (
        <Text numberOfLines={1} className="shrink font-mono text-[11px] text-muted-foreground">
          {footer.text}
        </Text>
      )}
      {footer && timestamp && <Text className="font-mono text-[11px] text-muted-foreground">·</Text>}
      {timestamp && <Text className="font-mono text-[11px] text-muted-foreground">{formatCalendarTime(embedTimestamp(timestamp), now)}</Text>}
    </View>
  );
};

// One embed as a card with its state on the leading edge, the way a topology node shows its status; its thumbnail above the text.
export const EmbedCard = ({ embed }: { embed: Embed }) => {
  const [open, setOpen] = useState(false);
  const fold = embedFold(embed);
  const foldLabel = embedFoldLabel(fold);
  const tone = embedCardTone(embed);
  const fields = open ? (embed.fields ?? []) : (embed.fields ?? []).slice(0, EMBED_FIELD_LIMIT);
  const image = httpUrl(embed.image?.url);
  const thumbnail = httpUrl(embed.thumbnail?.url);
  const hasFooter = !!(embed.footer || embed.timestamp);
  return (
    <View className="overflow-hidden rounded-lg border border-border bg-card">
      {tone && <View className={cn("absolute inset-y-0 left-0 w-[3px]", toneBg[tone])} />}
      <View className="gap-2.5 px-3.5 py-3">
        {thumbnail && <EmbedThumbnail uri={thumbnail} />}
        <EmbedHeading embed={embed} tone={tone} />
        {embed.description && (
          <View className={cn(fold.longDescription && !open && CLAMPED_DESCRIPTION)}>
            <DiscordMarkdown text={embed.description} textClassName="text-[13px] leading-5 text-foreground/80" />
          </View>
        )}
        {fields.length > 0 && <EmbedFieldGrid fields={fields} />}
        {image && <EmbedImage uri={image} />}
      </View>
      {(hasFooter || foldLabel) && (
        <View className="min-h-9 flex-row items-center justify-between gap-3 border-t border-border px-3.5 py-1">
          {hasFooter && <EmbedFooter embed={embed} />}
          {foldLabel && <FoldBar label={open ? "Show less" : foldLabel} open={open} onToggle={() => setOpen(!open)} />}
        </View>
      )}
    </View>
  );
};

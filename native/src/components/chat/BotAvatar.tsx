import { useState } from "react";
import { Image, View } from "react-native";
import { useCSSVariable } from "uniwind";

import { NexulMark } from "@/components/NexulMark";
import { useApiImageSource, useBotMediaSource } from "@/hooks/BotMediaHooks";

// assets/mark.svg, the default every bot starts with.
const NexulGlyph = ({ size }: { size: number }) => {
  const [ink] = useCSSVariable(["--color-muted-foreground"]);
  const color = ink === undefined ? "currentColor" : String(ink);
  return (
    <View style={{ width: size, height: size }} className="items-center justify-center rounded-full border border-border bg-card">
      <NexulMark size={size * 0.6} color={color} />
    </View>
  );
};

interface BotAvatarProps {
  src: string | undefined;
  name: string;
  size?: number;
}

// A picture that is missing or fails to load shows the Nexul glyph.
export const BotAvatar = ({ src, name, size = 20 }: BotAvatarProps) => {
  const served = useApiImageSource(src?.startsWith("/api/") ? src : undefined);
  const proxied = useBotMediaSource(src);
  const inline = src?.startsWith("data:image/") ? { uri: src } : undefined;
  const [failedSrc, setFailedSrc] = useState<string | undefined>(undefined);
  const source = failedSrc === src ? undefined : (served ?? proxied ?? inline);
  return (
    <>
      {!source && <NexulGlyph size={size} />}
      {source && (
        <Image
          accessibilityLabel={`Avatar of ${name}`}
          source={source}
          onError={() => setFailedSrc(src)}
          style={{ width: size, height: size }}
          className="rounded-full bg-accent"
        />
      )}
    </>
  );
};

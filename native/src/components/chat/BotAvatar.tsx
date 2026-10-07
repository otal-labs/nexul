import { useState } from "react";
import { Image, View, type ImageSourcePropType } from "react-native";
import Svg, { Circle, Path } from "react-native-svg";
import { useCSSVariable } from "uniwind";

import { httpUrl } from "@/models/Embed";
import { readSessionToken, useSessionStore } from "@/stores/sessionStore";

// assets/mark.svg, the default every bot starts with.
const NexulGlyph = () => {
  const [ink] = useCSSVariable(["--color-muted-foreground"]);
  const color = ink === undefined ? "currentColor" : String(ink);
  return (
    <View className="size-5 items-center justify-center rounded-full bg-accent">
      <Svg viewBox="0 0 24 24" width={14} height={14} fill="none" stroke={color} strokeWidth={2.25} strokeLinecap="round" strokeLinejoin="round">
        <Circle cx={12} cy={12} r={8.5} />
        <Path d="M7.6 7.6L12 12l4.4 4.4" />
        <Circle cx={12} cy={12} r={2.8} fill={color} stroke="none" />
        <Circle cx={6} cy={6} r={2.2} />
        <Circle cx={18} cy={18} r={2.2} />
      </Svg>
    </View>
  );
};

// The bot's own avatar is served under /api/ and needs the session; Android's Image only sends headers from an array source.
const avatarSource = (src: string | undefined, host: string | null): ImageSourcePropType | undefined => {
  if (src?.startsWith("/api/")) return [{ uri: `${host ?? ""}${src}`, headers: { Authorization: `Bearer ${readSessionToken() ?? ""}` } }];
  const direct = httpUrl(src) ?? (src?.startsWith("data:image/") ? src : undefined);
  return direct ? { uri: direct } : undefined;
};

interface BotAvatarProps {
  src: string | undefined;
  name: string;
}

// A picture that is missing or fails to load shows the Nexul glyph.
export const BotAvatar = ({ src, name }: BotAvatarProps) => {
  const host = useSessionStore((s) => s.host);
  const [failedSrc, setFailedSrc] = useState<string | undefined>(undefined);
  const source = failedSrc === src ? undefined : avatarSource(src, host);
  return (
    <>
      {!source && <NexulGlyph />}
      {source && (
        <Image
          accessibilityLabel={`Avatar of ${name}`}
          source={source}
          onError={() => setFailedSrc(src)}
          className="size-5 rounded-full bg-accent"
        />
      )}
    </>
  );
};

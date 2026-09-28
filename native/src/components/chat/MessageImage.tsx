import { useState } from "react";
import { Image, View } from "react-native";

import { Text } from "@/components/ui/text";
import { readSessionToken, useSessionStore } from "@/stores/sessionStore";

interface MessageImageProps {
  src: string;
  alt: string;
}

// The attachment route needs the bearer token, so the image request carries it as a header.
export const MessageImage = ({ src, alt }: MessageImageProps) => {
  const host = useSessionStore((s) => s.host);
  const [failed, setFailed] = useState(false);
  const [aspectRatio, setAspectRatio] = useState(4 / 3);
  return (
    <View className="py-1">
      {failed && (
        <View
          accessible
          role="img"
          aria-label={alt || "Image unavailable"}
          className="h-24 max-w-xs items-center justify-center rounded-md border border-dashed border-border"
        >
          <Text variant="muted">Image unavailable</Text>
        </View>
      )}
      {!failed && (
        <Image
          accessibilityLabel={alt || "Image"}
          source={{ uri: `${host ?? ""}${src}`, headers: { Authorization: `Bearer ${readSessionToken() ?? ""}` } }}
          resizeMode="contain"
          onError={() => setFailed(true)}
          onLoad={({ nativeEvent: { source } }) => setAspectRatio(source.width / source.height)}
          style={{ aspectRatio }}
          className="max-h-72 w-full max-w-xs rounded-md border border-border"
        />
      )}
    </View>
  );
};

import { createContext, useContext } from "react";
import type { ImageSourcePropType } from "react-native";

import { botMediaPath } from "@/models/Embed";
import { readSessionToken, useSessionStore } from "@/stores/sessionStore";

// Set around a bot message so its images, however deep, ask the media proxy on that message's behalf.
export const BotMessageIdContext = createContext<string | undefined>(undefined);

// An /api/ path with the session; Android's Image only sends headers from an array source.
export const useApiImageSource = (path: string | undefined): ImageSourcePropType | undefined => {
  const host = useSessionStore((s) => s.host);
  if (!path) return undefined;
  return [{ uri: `${host ?? ""}${path}`, headers: { Authorization: `Bearer ${readSessionToken() ?? ""}` } }];
};

// The proxied source of an external image the enclosing bot message shows; undefined outside one or for a non-http(s) URL.
export const useBotMediaSource = (raw: string | undefined): ImageSourcePropType | undefined => {
  const messageId = useContext(BotMessageIdContext);
  return useApiImageSource(messageId ? botMediaPath(messageId, raw) : undefined);
};

import { createContext, useContext } from "react";
import { PixelRatio, type ImageSourcePropType } from "react-native";

import { botMediaPath } from "@nexul/client-core/embed";

import { readSessionToken, useSessionStore } from "@/stores/sessionStore";

// Set around a bot message so its images, however deep, ask the media proxy on that message's behalf.
export const BotMessageIdContext = createContext<string | undefined>(undefined);

// A still PNG sized to the avatar's box, never the upload: Android animates an uploaded GIF on the UI thread per copy.
export const avatarPath = (path: string, points: number) =>
  `${path}${path.includes("?") ? "&" : "?"}size=${Math.ceil(points * PixelRatio.get())}`;

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

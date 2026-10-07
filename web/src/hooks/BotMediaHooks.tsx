import { createContext, useContext } from "react";

import { botMediaPath } from "@/models/Embed";

// Set around a bot message so its images, however deep, ask the media proxy on that message's behalf.
export const BotMessageIdContext = createContext<string | undefined>(undefined);

// The proxy path of an external image the enclosing bot message shows; undefined outside one or for a non-http(s) URL.
export const useBotMediaPath = (raw: string | undefined): string | undefined => {
  const messageId = useContext(BotMessageIdContext);
  return messageId ? botMediaPath(messageId, raw) : undefined;
};

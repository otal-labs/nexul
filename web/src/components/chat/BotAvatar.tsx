import { useState } from "react";

import { Logo } from "@/components/Logo";
import { useAttachmentBlob } from "@/hooks/AttachmentHooks";
import { useBotMediaPath } from "@/hooks/BotMediaHooks";

// A picture that is missing or fails to load shows the Nexul glyph, the default every bot starts with, toned like a person's fill.
const BotAvatarImage = ({ src }: { src: string | undefined }) => {
  const [failedSrc, setFailedSrc] = useState<string | undefined>(undefined);
  const shown = !!src && failedSrc !== src;
  return (
    <>
      {shown && <img src={src} alt="" onError={() => setFailedSrc(src)} className="size-8 shrink-0 rounded-full bg-accent object-cover" />}
      {!shown && <Logo className="rounded-full bg-muted text-muted-foreground [&_svg]:size-4.5" />}
    </>
  );
};
// The bot's own avatar (under /api/) and a sender's override (through the media proxy) need the session, so use blobs.
const BlobBotAvatar = ({ path }: { path: string }) => {
  const { data: blobUrl } = useAttachmentBlob(path);
  return <BotAvatarImage src={blobUrl} />;
};

export const BotAvatar = ({ src }: { src: string | undefined }) => {
  const proxied = useBotMediaPath(src);
  const path = src?.startsWith("/api/") ? src : proxied;
  return (
    <>
      {path && <BlobBotAvatar path={path} />}
      {!path && <BotAvatarImage src={src?.startsWith("data:image/") ? src : undefined} />}
    </>
  );
};

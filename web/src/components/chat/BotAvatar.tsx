import { useState } from "react";

import { useAttachmentBlob } from "@/hooks/AttachmentHooks";
import { httpUrl } from "@/models/Embed";

const GLYPH = "/favicon.svg";

// A picture that is missing or fails to load shows the Nexul glyph, the default every bot starts with.
const BotAvatarImage = ({ src }: { src: string | undefined }) => {
  const [failedSrc, setFailedSrc] = useState<string | undefined>(undefined);
  const shown = src && failedSrc !== src ? src : GLYPH;
  return (
    <img
      src={shown}
      alt=""
      referrerPolicy="no-referrer"
      onError={() => setFailedSrc(src)}
      className="size-6 shrink-0 rounded-full bg-accent object-cover"
    />
  );
};

// The bot's own avatar is served under /api/ and needs the session, so it loads through the blob cache.
const ServedBotAvatar = ({ src }: { src: string }) => {
  const { data: blobUrl } = useAttachmentBlob(src);
  return <BotAvatarImage src={blobUrl} />;
};

export const BotAvatar = ({ src }: { src: string | undefined }) => {
  const served = src?.startsWith("/api/") === true;
  const direct = httpUrl(src) ?? (src?.startsWith("data:image/") ? src : undefined);
  return (
    <>
      {served && <ServedBotAvatar src={src ?? ""} />}
      {!served && <BotAvatarImage src={direct} />}
    </>
  );
};

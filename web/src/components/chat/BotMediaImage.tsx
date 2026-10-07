import { useAttachmentBlob } from "@/hooks/AttachmentHooks";
import { useBotMediaPath } from "@/hooks/BotMediaHooks";

// An embed's picture through the media proxy; one that is missing or fails to load shows nothing, as Discord does.
export const BotMediaImage = ({ url, className }: { url: string | undefined; className: string }) => {
  const { data: src } = useAttachmentBlob(useBotMediaPath(url) ?? null);
  if (!src) return null;
  return <img src={src} alt="" className={className} />;
};

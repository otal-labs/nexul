import { useState } from "react";

import { useAttachmentBlob } from "@/hooks/AttachmentHooks";
import { cn, initials } from "@/lib/utils";

interface PersonAvatarProps {
  login: string;
  // The directory's avatar_url; without one, github.com/<login>.png stands in.
  src?: string | undefined;
  className?: string | undefined;
  /** The hover text; the login when omitted. */
  label?: string | undefined;
}

// The initials circle covers a picture that is missing, still loading, or 404s.
const AvatarImage = ({ login, src, className, label = login }: PersonAvatarProps) => {
  const [failedSrc, setFailedSrc] = useState<string | undefined>(undefined);
  const showImage = !!src && failedSrc !== src;

  return (
    <>
      {!showImage && (
        <span
          title={label}
          className={cn(
            "flex size-5 shrink-0 items-center justify-center rounded-full bg-accent text-[9px] font-semibold text-accent-foreground",
            className,
          )}
        >
          {initials(login)}
        </span>
      )}
      {showImage && (
        <img
          src={src}
          alt=""
          title={label}
          referrerPolicy="no-referrer"
          onError={() => setFailedSrc(src)}
          className={cn("size-5 shrink-0 rounded-full bg-accent object-cover", className)}
        />
      )}
    </>
  );
};

// An uploaded picture is served under /api/ and needs the session, so it loads through the blob cache, keyed by its versioned URL.
const ServedAvatar = ({ login, src, className, label }: PersonAvatarProps) => {
  const { data: blobUrl } = useAttachmentBlob(src ?? null);
  return <AvatarImage login={login} src={blobUrl} className={className} label={label} />;
};

export const PersonAvatar = ({ login, src, className, label }: PersonAvatarProps) => {
  const served = src?.startsWith("/api/") === true;
  return (
    <>
      {served && <ServedAvatar login={login} src={src} className={className} label={label} />}
      {!served && (
        <AvatarImage login={login} src={src || `https://github.com/${login}.png`} className={className} label={label} />
      )}
    </>
  );
};

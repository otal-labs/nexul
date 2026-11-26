import { useState } from "react";

import { useAttachmentBlob } from "@/hooks/AttachmentHooks";
import { avatarFill } from "@nexul/client-core/avatarFill";
import { cn, initials } from "@/lib/utils";

interface PersonAvatarProps {
  login: string;
  // The directory's avatar_url; without one, github.com/<login>.png stands in.
  src?: string | undefined;
  className?: string | undefined;
  /** The hover text; the login when omitted. */
  label?: string | undefined;
  /** The display name, whose initials stand in for a missing picture; the login's when empty. Required, so no caller forgets it. */
  name: string;
}

// The flat fill shows until the picture has loaded, so a slow one never paints a broken image, and takes the initials
// when there is no picture or it 404s; under 24px it keeps one initial at 10px.
const AvatarImage = ({ login, src, className, label = login, name }: PersonAvatarProps) => {
  const [failedSrc, setFailedSrc] = useState<string | undefined>(undefined);
  const [loadedSrc, setLoadedSrc] = useState<string | undefined>(undefined);
  const showImage = !!src && failedSrc !== src;
  const loaded = showImage && loadedSrc === src;
  const [first = "", second = ""] = initials(name || login);

  return (
    <span
      title={showImage ? undefined : label}
      style={loaded ? undefined : { backgroundColor: avatarFill(login) }}
      className={cn(
        "@container relative flex size-5 shrink-0 items-center justify-center rounded-full font-semibold text-white select-none",
        className,
      )}
    >
      {!showImage && (
        <span className="text-[max(var(--text-initial),38cqw)] leading-none @min-[1.5rem]:text-[max(var(--text-micro),38cqw)]">
          {first}
          <span className="@max-[1.5rem]:hidden">{second}</span>
        </span>
      )}
      {showImage && (
        <img
          src={src}
          alt=""
          title={label}
          referrerPolicy="no-referrer"
          onLoad={() => setLoadedSrc(src)}
          onError={() => setFailedSrc(src)}
          className="absolute inset-0 size-full rounded-[inherit] object-cover"
        />
      )}
    </span>
  );
};

// An uploaded picture is served under /api/ and needs the session, so it loads through the blob cache, keyed by its versioned URL.
const ServedAvatar = ({ login, src, className, label, name }: PersonAvatarProps) => {
  const { data: blobUrl } = useAttachmentBlob(src ?? null);
  return <AvatarImage login={login} src={blobUrl} className={className} label={label} name={name} />;
};

export const PersonAvatar = ({ login, src, className, label, name }: PersonAvatarProps) => {
  const served = src?.startsWith("/api/") === true;
  return (
    <>
      {served && <ServedAvatar login={login} src={src} className={className} label={label} name={name} />}
      {!served && (
        <AvatarImage login={login} src={src || `https://github.com/${login}.png`} className={className} label={label} name={name} />
      )}
    </>
  );
};

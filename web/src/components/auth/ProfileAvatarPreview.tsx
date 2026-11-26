import { PersonAvatar } from "@/components/PersonAvatar";
import { avatarFill } from "@nexul/client-core/avatarFill";
import { cn, initials } from "@/lib/utils";

interface ProfileAvatarPreviewProps {
  src: string;
  login: string;
  name: string;
  className?: string;
}

// Without a picture the profile shows the fill everyone else sees, its initials following the name as it is typed;
// a served picture loads with the session through PersonAvatar, painting that fill until it arrives.
export const ProfileAvatarPreview = ({ src, login, name, className = "size-16 text-xl" }: ProfileAvatarPreviewProps) => (
  <>
    {src && <PersonAvatar src={src} login={login} name={name.trim()} label={name.trim() || login} className={className} />}
    {!src && (
      <span
        aria-hidden
        style={{ backgroundColor: avatarFill(login) }}
        className={cn("flex shrink-0 items-center justify-center rounded-full font-semibold text-white select-none", className)}
      >
        {initials(name.trim() || login)}
      </span>
    )}
  </>
);

import { avatarGradient } from "@/lib/avatarGradient";
import { cn, initials } from "@/lib/utils";

interface ProfileAvatarPreviewProps {
  src: string;
  login: string;
  name: string;
  className?: string;
}

// Without a picture the profile shows the gradient everyone else sees, its initials following the name as it is typed.
export const ProfileAvatarPreview = ({ src, login, name, className = "size-16 text-xl" }: ProfileAvatarPreviewProps) => (
  <>
    {src && <img src={src} alt="" className={cn("rounded-full object-cover", className)} referrerPolicy="no-referrer" />}
    {!src && (
      <span
        aria-hidden
        style={{ backgroundImage: avatarGradient(login) }}
        className={cn("flex shrink-0 items-center justify-center rounded-full font-semibold text-white select-none [text-shadow:0_1px_2px_oklch(0_0_0/0.35)]", className)}
      >
        {initials(name.trim() || login)}
      </span>
    )}
  </>
);

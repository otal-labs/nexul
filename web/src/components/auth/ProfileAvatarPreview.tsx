import { avatarGradient } from "@/lib/avatarGradient";
import { initials } from "@/lib/utils";

interface ProfileAvatarPreviewProps {
  src: string;
  login: string;
  name: string;
}

// Without a picture the profile shows the gradient everyone else sees, its initials following the name as it is typed.
export const ProfileAvatarPreview = ({ src, login, name }: ProfileAvatarPreviewProps) => (
  <>
    {src && <img src={src} alt="" className="size-16 rounded-full object-cover" referrerPolicy="no-referrer" />}
    {!src && (
      <span
        aria-hidden
        style={{ backgroundImage: avatarGradient(login) }}
        className="flex size-16 shrink-0 items-center justify-center rounded-full text-xl font-semibold text-white select-none [text-shadow:0_1px_2px_oklch(0_0_0/0.35)]"
      >
        {initials(name.trim() || login)}
      </span>
    )}
  </>
);

import { ProfileAvatarPreview } from "@/components/auth/ProfileAvatarPreview";
import { FormInput } from "@/components/FormInput";
import { Button } from "@/components/ui/button";
import { useProfileForm } from "@/hooks/useProfileForm";
import { cn } from "@/lib/utils";
import type { User } from "@/models/User";

interface ProfileFormProps {
  user: User;
  submitLabel: string;
  submitClassName?: string;
  onSaved?: () => void;
}

// The first-login wizard's stacked form; Settings → Profile lays the same fields out as rows (ProfileSettingsForm).
export const ProfileForm = ({ user, submitLabel, submitClassName, onSaved }: ProfileFormProps) => {
  const { form, fileInputRef, name, avatarOverrideUrl, previewSrc, avatarError, onFileChange, removeAvatar, submit } =
    useProfileForm(user, onSaved);

  return (
    <form onSubmit={(e) => void submit(e)} className="space-y-6">
      <div className="flex items-center gap-4">
        <ProfileAvatarPreview src={previewSrc} login={user.login} name={name} />
        <div className="flex gap-2">
          <Button type="button" variant="outline" size="sm" onClick={() => fileInputRef.current?.click()}>
            Upload image
          </Button>
          <Button type="button" variant="ghost" size="sm" onClick={removeAvatar} disabled={!avatarOverrideUrl}>
            Remove
          </Button>
        </div>
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          aria-label="Upload avatar image"
          className="hidden"
          onChange={(e) => void onFileChange(e)}
        />
      </div>
      {avatarError && (
        <p role="alert" className="text-sm text-destructive">
          {avatarError}
        </p>
      )}
      <FormInput control={form.control} name="name" label="Display name" placeholder={user.login} />
      <Button type="submit" className={cn(submitClassName)} loading={form.formState.isSubmitting}>
        {submitLabel}
      </Button>
    </form>
  );
};

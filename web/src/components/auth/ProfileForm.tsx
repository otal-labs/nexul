import { useId } from "react";

import { ProfileAvatarPreview } from "@/components/auth/ProfileAvatarPreview";
import { Button } from "@/components/ui/button";
import { FormInput } from "@/components/FormInput";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { useProfileForm } from "@/hooks/useProfileForm";
import type { User } from "@/models/User";

interface ProfileFormProps {
  user: User;
  submitLabel: string;
  onSaved?: () => void;
  onBack?: (() => void) | undefined;
}

// The wizards' stacked form; Settings → Profile lays the same fields out as rows (ProfileSettingsForm).
export const ProfileForm = ({ user, submitLabel, onSaved, onBack }: ProfileFormProps) => {
  const formId = useId();
  const { form, fileInputRef, name, avatarOverrideUrl, previewSrc, avatarError, onFileChange, removeAvatar, submit } =
    useProfileForm(user, onSaved);

  return (
    <form id={formId} onSubmit={(e) => void submit(e)} className="space-y-6">
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
      <FormInput control={form.control} name="name" label="Display name" placeholder={user.login} className="max-w-80" />
      <WizardFooter onBack={onBack}>
        <Button type="submit" form={formId} loading={form.formState.isSubmitting}>
          {submitLabel}
        </Button>
      </WizardFooter>
    </form>
  );
};

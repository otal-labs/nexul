import { ProfileAvatarPreview } from "@/components/auth/ProfileAvatarPreview";
import { FormInput } from "@/components/FormInput";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SettingsRow, SettingsRows } from "@/components/settings/SettingsRow";
import { SettingsSaveBar } from "@/components/settings/SettingsSaveBar";
import { Button } from "@/components/ui/button";
import { useFlash } from "@/hooks/useFlash";
import { useProfileForm } from "@/hooks/useProfileForm";
import type { User } from "@/models/User";

const FORM_ID = "profile-form";

export const ProfileCard = ({ user }: { user: User }) => {
  const [saved, flash] = useFlash();
  const { form, fileInputRef, name, avatarOverrideUrl, previewSrc, avatarError, onFileChange, removeAvatar, discard, submit } =
    useProfileForm(user, flash);

  return (
    <SettingsCard
      id="profile"
      title="Profile"
      description="How everyone in your workspaces sees you."
      footer={
        <SettingsSaveBar
          form={FORM_ID}
          dirty={form.formState.isDirty}
          saving={form.formState.isSubmitting}
          saved={saved}
          onDiscard={discard}
        />
      }
    >
      <form id={FORM_ID} onSubmit={(e) => void submit(e)}>
        <SettingsRows>
          <SettingsRow label="Picture" description="Without one, your sign-in account's picture is used.">
            <div className="flex items-center gap-3">
              <ProfileAvatarPreview src={previewSrc} login={user.login} name={name} className="size-11 text-base" />
              <Button type="button" variant="outline" size="sm" onClick={() => fileInputRef.current?.click()}>
                Upload
              </Button>
              <Button type="button" variant="ghost" size="sm" onClick={removeAvatar} disabled={!avatarOverrideUrl}>
                Remove
              </Button>
              <input
                ref={fileInputRef}
                type="file"
                accept="image/*"
                aria-label="Upload picture"
                className="hidden"
                onChange={(e) => void onFileChange(e)}
              />
            </div>
          </SettingsRow>
          {avatarError && (
            <p role="alert" className="py-3 text-sm text-destructive">
              {avatarError}
            </p>
          )}
          <SettingsRow label="Display name" description="Mentions and the team list use it." htmlFor="display-name">
            <div className="w-full">
              <FormInput control={form.control} name="name" id="display-name" label="Display name" hideLabel placeholder={user.login} />
            </div>
          </SettingsRow>
        </SettingsRows>
      </form>
    </SettingsCard>
  );
};

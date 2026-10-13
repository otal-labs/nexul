import { ProfileAvatarPreview } from "@/components/auth/ProfileAvatarPreview";
import { FormInput } from "@/components/FormInput";
import { Button } from "@/components/ui/button";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useProfileForm } from "@/hooks/useProfileForm";
import type { User } from "@/models/User";

import { StepFooter, useFlow } from "../flow";

const FORM_ID = "welcome-profile";

const ProfileFields = ({ user }: { user: User }) => {
  const flow = useFlow();
  const { form, fileInputRef, name, avatarOverrideUrl, previewSrc, avatarError, onFileChange, removeAvatar, submit } = useProfileForm(
    user,
    () => flow.next("done"),
  );
  return (
    <>
      <form id={FORM_ID} onSubmit={(e) => void submit(e)} className="space-y-6">
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
          <input ref={fileInputRef} type="file" accept="image/*" aria-label="Upload avatar image" className="hidden" onChange={(e) => void onFileChange(e)} />
        </div>
        {avatarError && (
          <p role="alert" className="text-sm text-destructive">
            {avatarError}
          </p>
        )}
        <FormInput control={form.control} name="name" label="Display name" placeholder={user.login} />
      </form>
      {/* Profile is never skipped: the sign-in already filled it, so Continue with nothing changed is the skip. */}
      <StepFooter skip={false} primary={{ label: "Continue", form: FORM_ID }} />
    </>
  );
};

export const ProfileStep = () => {
  const { data } = useFetchMe();
  return <>{data && <ProfileFields user={data.user} />}</>;
};

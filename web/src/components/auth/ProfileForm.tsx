import { zodResolver } from "@hookform/resolvers/zod";
import { useRef, type ChangeEvent } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";

import { errorMessage } from "@/api/client";
import { ProfileAvatarPreview } from "@/components/auth/ProfileAvatarPreview";
import { FormInput } from "@/components/FormInput";
import { Button } from "@/components/ui/button";
import { useUpdateProfile } from "@/hooks/AuthHooks";
import { cn } from "@/lib/utils";
import type { User } from "@/models/User";
import { readAvatarFile } from "@/utils/AvatarFileUtility";

const ProfileFormSchema = z.object({
  name: z.string().trim().min(1, "Display name is required"),
  avatarOverrideUrl: z.string(),
});

type ProfileFormData = z.infer<typeof ProfileFormSchema>;

interface ProfileFormProps {
  user: User;
  submitLabel: string;
  submitClassName?: string;
  onSaved?: () => void;
}

// The one display name and picture override form, shared by the first-login wizard and Your settings → Profile.
export const ProfileForm = ({ user, submitLabel, submitClassName, onSaved }: ProfileFormProps) => {
  const updateProfile = useUpdateProfile();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const form = useForm<ProfileFormData>({
    defaultValues: {
      // Effective name/avatar (override if set, else provider-sourced), so untouched fields round-trip.
      name: user.display_name || user.name,
      avatarOverrideUrl: user.avatar_override_url ?? "",
    },
    resolver: zodResolver(ProfileFormSchema),
  });

  const avatarOverrideUrl = useWatch({ control: form.control, name: "avatarOverrideUrl" });
  const name = useWatch({ control: form.control, name: "name" });
  const avatarError = form.formState.errors.avatarOverrideUrl?.message as string | undefined;
  const previewSrc = avatarOverrideUrl || user.avatar_url;

  const onFileChange = async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const read = await readAvatarFile(file);
    if ("error" in read) {
      form.setError("avatarOverrideUrl", { type: "manual", message: read.error });
      return;
    }
    form.clearErrors("avatarOverrideUrl");
    form.setValue("avatarOverrideUrl", read.dataUrl, { shouldDirty: true });
  };

  const onRemoveAvatar = () => {
    form.clearErrors("avatarOverrideUrl");
    form.setValue("avatarOverrideUrl", "", { shouldDirty: true });
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  const onSubmit = async (data: ProfileFormData) => {
    try {
      await updateProfile.mutateAsync({
        display_name: data.name,
        avatar_override_url: data.avatarOverrideUrl,
      });
      onSaved?.();
    } catch (error) {
      // Surface backend validation on the avatar field, not just a generic toast (the hook toasts too).
      form.setError("avatarOverrideUrl", { type: "server", message: errorMessage(error) });
    }
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
      <div className="flex items-center gap-4">
        <ProfileAvatarPreview src={previewSrc} login={user.login} name={name} />
        <div className="flex flex-col gap-2">
          <div className="flex gap-2">
            <Button type="button" variant="outline" size="sm" onClick={() => fileInputRef.current?.click()}>
              Upload image
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={onRemoveAvatar}
              disabled={!avatarOverrideUrl}
            >
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

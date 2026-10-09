import { zodResolver } from "@hookform/resolvers/zod";
import { useRef, type ChangeEvent } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";

import { errorMessage } from "@/api/client";
import { useUpdateProfile } from "@/hooks/AuthHooks";
import type { User } from "@/models/User";
import { readAvatarFile } from "@/utils/AvatarFileUtility";

const ProfileFormSchema = z.object({
  name: z.string().trim().min(1, "Display name is required"),
  avatarOverrideUrl: z.string(),
});

type ProfileFormData = z.infer<typeof ProfileFormSchema>;

// The display name and picture override form, shared by the first-login wizard and Settings → Profile.
export const useProfileForm = (user: User, onSaved?: () => void) => {
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

  const clearFileInput = () => {
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  const removeAvatar = () => {
    form.clearErrors("avatarOverrideUrl");
    form.setValue("avatarOverrideUrl", "", { shouldDirty: true });
    clearFileInput();
  };

  const discard = () => {
    form.reset();
    clearFileInput();
  };

  const submit = form.handleSubmit(async (data) => {
    try {
      await updateProfile.mutateAsync({ display_name: data.name, avatar_override_url: data.avatarOverrideUrl });
      form.reset(data);
      onSaved?.();
    } catch (error) {
      // Surface backend validation on the avatar field, not just a generic toast (the hook toasts too).
      form.setError("avatarOverrideUrl", { type: "server", message: errorMessage(error) });
    }
  });

  return {
    form,
    fileInputRef,
    name,
    avatarOverrideUrl,
    previewSrc: avatarOverrideUrl || user.avatar_url,
    avatarError: form.formState.errors.avatarOverrideUrl?.message,
    onFileChange,
    removeAvatar,
    discard,
    submit,
  };
};

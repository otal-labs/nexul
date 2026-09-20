import { zodResolver } from "@hookform/resolvers/zod";
import { useRef, type ChangeEvent } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";

import { errorMessage } from "@/api/client";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { FormInput } from "@/components/FormInput";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { useFetchMe, useUpdateProfile } from "@/hooks/AuthHooks";
import type { User } from "@/models/User";

// Mirrors the backend's maxAvatarOverrideBytes so an oversized image is rejected before the round-trip.
const MAX_AVATAR_BYTES = 10 * 1024 * 1024;

const IntroduceYourselfFormSchema = z.object({
  name: z.string().trim().min(1, "Name is required"),
  avatarOverrideUrl: z.string(),
});

type IntroduceYourselfFormData = z.infer<typeof IntroduceYourselfFormSchema>;

interface IntroduceYourselfStepProps {
  onContinue: () => void;
}

// Fetches its own user data, like SetupWorkspaceStep, so it only needs onContinue as a prop.
export const IntroduceYourselfStep = ({ onContinue }: IntroduceYourselfStepProps) => {
  const { data, isPending, error } = useFetchMe();

  return (
    <div className="space-y-6">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && <IntroduceYourselfForm user={data.user} onContinue={onContinue} />}
    </div>
  );
};

interface IntroduceYourselfFormProps {
  user: User;
  onContinue: () => void;
}

const IntroduceYourselfForm = ({ user, onContinue }: IntroduceYourselfFormProps) => {
  const updateProfile = useUpdateProfile();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const form = useForm<IntroduceYourselfFormData>({
    defaultValues: {
      // Effective name/avatar (override if set, else provider-sourced), so untouched fields round-trip.
      name: user.display_name || user.name,
      avatarOverrideUrl: user.avatar_override_url ?? "",
    },
    resolver: zodResolver(IntroduceYourselfFormSchema),
  });

  const avatarOverrideUrl = useWatch({ control: form.control, name: "avatarOverrideUrl" });
  const avatarError = form.formState.errors.avatarOverrideUrl?.message as string | undefined;
  const previewSrc = avatarOverrideUrl || user.avatar_url;

  const onFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    if (!file.type.startsWith("image/")) {
      form.setError("avatarOverrideUrl", { type: "manual", message: "Please choose an image file" });
      return;
    }
    if (file.size > MAX_AVATAR_BYTES) {
      form.setError("avatarOverrideUrl", { type: "manual", message: "Image must be under 10MB" });
      return;
    }

    const reader = new FileReader();
    reader.onload = () => {
      form.clearErrors("avatarOverrideUrl");
      form.setValue("avatarOverrideUrl", reader.result as string, { shouldDirty: true });
    };
    reader.readAsDataURL(file);
  };

  const onRemoveAvatar = () => {
    form.clearErrors("avatarOverrideUrl");
    form.setValue("avatarOverrideUrl", "", { shouldDirty: true });
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  const onSubmit = async (data: IntroduceYourselfFormData) => {
    try {
      await updateProfile.mutateAsync({
        display_name: data.name,
        avatar_override_url: data.avatarOverrideUrl,
      });
      onContinue();
    } catch (error) {
      // Surface backend validation on the avatar field, not just a generic toast (the hook toasts too).
      form.setError("avatarOverrideUrl", { type: "server", message: errorMessage(error) });
    }
  };

  return (
    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
      <div className="flex items-center gap-4">
        {previewSrc && (
          <img
            src={previewSrc}
            alt=""
            className="size-16 rounded-full object-cover"
            referrerPolicy="no-referrer"
          />
        )}
        {!previewSrc && <div className="size-16 rounded-full bg-muted" />}
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
            onChange={onFileChange}
          />
        </div>
      </div>
      {avatarError && (
        <p role="alert" className="text-sm text-destructive">
          {avatarError}
        </p>
      )}
      <FormInput control={form.control} name="name" label="Name" placeholder={user.login} />
      <Button type="submit" className="w-full" disabled={form.formState.isSubmitting}>
        {form.formState.isSubmitting ? "Saving…" : "Continue"}
      </Button>
    </form>
  );
};

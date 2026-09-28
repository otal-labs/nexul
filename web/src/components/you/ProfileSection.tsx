import { ProfileForm } from "@/components/auth/ProfileForm";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useFetchMe } from "@/hooks/AuthHooks";

export const ProfileSection = () => {
  const { data, isPending, error } = useFetchMe();

  return (
    <SettingsCard
      id="profile"
      title="Profile"
      description="The name and picture everyone in your workspaces sees. Leave the picture out to use your sign-in account's."
    >
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && <ProfileForm user={data.user} submitLabel="Save" />}
    </SettingsCard>
  );
};

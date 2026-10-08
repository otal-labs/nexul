import { ProfileForm } from "@/components/auth/ProfileForm";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { SignInAccountsSection } from "@/components/you/SignInAccountsSection";
import { useFetchMe } from "@/hooks/AuthHooks";

export const ProfileSection = () => {
  const { data, isPending, error } = useFetchMe();

  return (
    <>
      <SettingsCard
        id="profile"
        title="Profile"
        description="Shown to everyone in your workspaces. With no picture, your sign-in account's is used."
      >
        {isPending && <LoadingDisplay />}
        {error && <ErrorDisplay error={error} />}
        {data && <ProfileForm user={data.user} submitLabel="Save" />}
      </SettingsCard>
      <SignInAccountsSection />
    </>
  );
};

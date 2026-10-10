import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { GitHubAccessSection } from "@/components/you/GitHubAccessSection";
import { ProfileCard } from "@/components/you/ProfileCard";
import { SignInAccountsSection } from "@/components/you/SignInAccountsSection";
import { useFetchMe } from "@/hooks/AuthHooks";

export const ProfileSection = () => {
  const { data, isPending, error } = useFetchMe();

  return (
    <>
      {isPending && (
        <SettingsCard id="profile" title="Profile">
          <LoadingDisplay />
        </SettingsCard>
      )}
      {error && (
        <SettingsCard id="profile" title="Profile">
          <ErrorDisplay error={error} />
        </SettingsCard>
      )}
      {data && <ProfileCard user={data.user} />}
      <SignInAccountsSection />
      <GitHubAccessSection />
    </>
  );
};

import { ProfileForm } from "@/components/auth/ProfileForm";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchMe } from "@/hooks/AuthHooks";
import { useWelcomeFlow } from "@/hooks/useWelcomeFlow";

// No Skip: the sign-in already filled the profile, so Continue with nothing changed is the skip.
export const ProfileStep = () => {
  const { data: me, isPending, error } = useFetchMe();
  const { next } = useWelcomeFlow();
  return (
    <>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {me && <ProfileForm user={me.user} submitLabel="Continue" onSaved={next} />}
    </>
  );
};

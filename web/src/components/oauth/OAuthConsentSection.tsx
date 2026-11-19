import { errorMessage } from "@/api/client";
import { InlineNotice } from "@/components/InlineNotice";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { OAuthConsentPanel } from "@/components/oauth/OAuthConsentPanel";
import { useDecideOAuth, useFetchOAuthConsent } from "@/hooks/OAuthHooks";
import { isFollowableRedirect, redirectTarget } from "@/models/OAuth";

interface OAuthConsentSectionProps {
  search: string;
}

export const OAuthConsentSection = ({ search }: OAuthConsentSectionProps) => {
  const { data: consent, error, isPending } = useFetchOAuthConsent(search);
  const decide = useDecideOAuth(search);

  const onDecide = (approve: boolean) =>
    decide.mutate(approve, {
      onSuccess: ({ redirect_to }) => {
        if (isFollowableRedirect(redirect_to)) window.location.assign(redirect_to);
      },
    });

  return (
    <section aria-label="Sign an app in">
      {isPending && <LoadingDisplay label="Checking the app…" />}
      {error && <InlineNotice title="This sign-in can't go ahead">{errorMessage(error)}</InlineNotice>}
      {consent && !decide.isSuccess && <OAuthConsentPanel consent={consent} pending={decide.isPending} onDecide={onDecide} />}
      {consent && decide.isSuccess && (
        <InlineNotice tone="info" title={`Sent back to ${redirectTarget(consent.redirect_uri)}`}>You can close this tab.</InlineNotice>
      )}
      {decide.error && <InlineNotice title="Couldn't send your answer" className="mt-4">{errorMessage(decide.error)}</InlineNotice>}
    </section>
  );
};

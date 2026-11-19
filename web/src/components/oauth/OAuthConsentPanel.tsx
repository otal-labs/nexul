import { SignedOutHeader } from "@/components/auth/SignedOutHeader";
import { Button } from "@/components/ui/button";
import { useProductName } from "@/ee/hooks/useProductName";
import { redirectTarget, type OAuthConsent } from "@/models/OAuth";

interface OAuthConsentPanelProps {
  consent: OAuthConsent;
  pending: boolean;
  onDecide: (approve: boolean) => void;
}

// The app names itself when it registers, so where approving sends you is shown beside the name it chose.
export const OAuthConsentPanel = ({ consent, pending, onDecide }: OAuthConsentPanelProps) => {
  const name = useProductName();
  return (
    <div>
      <SignedOutHeader
        title={`Allow ${consent.client_name}?`}
        subtitle={`It will use ${name} as you, with your permissions and nothing more, until you revoke it under Settings, Security.`}
      />
      <p className="mt-4 text-row text-muted-foreground">
        Allowing sends you back to <span className="font-mono text-foreground break-all">{redirectTarget(consent.redirect_uri)}</span>.
      </p>
      <div className="mt-5 grid gap-1.5">
        <Button type="button" size="lg" className="w-full" loading={pending} onClick={() => onDecide(true)}>Allow</Button>
        <Button type="button" variant="ghost" className="w-full text-muted-foreground" disabled={pending} onClick={() => onDecide(false)}>Deny</Button>
      </div>
    </div>
  );
};

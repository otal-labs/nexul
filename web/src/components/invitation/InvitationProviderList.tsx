import { InvitationProviderButton } from "@/components/invitation/InvitationProviderButton";
import type { InvitationProvider } from "@/models/Invitation";

interface InvitationProviderListProps {
  providers: InvitationProvider[];
  disabled: boolean;
  onSelect: (provider: InvitationProvider) => void;
}

export const InvitationProviderList = ({ providers, disabled, onSelect }: InvitationProviderListProps) => (
  <div className="space-y-3">
    <p className="text-center text-sm text-muted-foreground">Sign in to accept.</p>
    {providers.map((provider) => <InvitationProviderButton key={provider} provider={provider} disabled={disabled} onSelect={onSelect} />)}
  </div>
);

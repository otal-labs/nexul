import { InvitationProviderButton } from "@/components/invitation/InvitationProviderButton";
import type { InvitationProvider } from "@/models/Invitation";

interface InvitationProviderListProps {
  providers: InvitationProvider[];
  disabled: boolean;
  onSelect: (provider: InvitationProvider) => void;
}

export const InvitationProviderList = ({ providers, disabled, onSelect }: InvitationProviderListProps) => (
  <div className="mt-7 space-y-3">
    <p className="text-center text-[13px] text-muted-foreground">Sign in to accept</p>
    <div className="grid gap-2">
      {providers.map((provider, index) => <InvitationProviderButton key={provider} provider={provider} primary={index === 0} disabled={disabled} onSelect={onSelect} />)}
    </div>
  </div>
);

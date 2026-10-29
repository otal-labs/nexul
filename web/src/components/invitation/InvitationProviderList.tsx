import { ProviderButton } from "@/components/auth/ProviderButton";
import type { InvitationProvider } from "@/models/Invitation";

interface InvitationProviderListProps {
  providers: InvitationProvider[];
  disabled: boolean;
  onSelect: (provider: InvitationProvider) => void;
}

// The first provider the owner turned on takes the ember, the rest are outline.
export const InvitationProviderList = ({ providers, disabled, onSelect }: InvitationProviderListProps) => (
  <div className="mt-4 space-y-2">
    <p className="text-[13px] text-muted-foreground">Sign in to accept</p>
    <div className="grid gap-2">
      {providers.map((provider, index) => <ProviderButton key={provider} provider={provider} primary={index === 0} disabled={disabled} onSelect={onSelect} />)}
    </div>
  </div>
);

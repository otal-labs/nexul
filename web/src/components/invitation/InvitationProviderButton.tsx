import { DiscordMark, GithubMark, GoogleMark } from "@/components/ProviderMarks";
import { Button } from "@/components/ui/button";
import type { InvitationProvider } from "@/models/Invitation";

const labels: Record<InvitationProvider, string> = { github: "GitHub", google: "Google", discord: "Discord" };

const ProviderIcon = ({ provider }: { provider: InvitationProvider }) => {
  const icons: Record<InvitationProvider, typeof GithubMark> = { github: GithubMark, google: GoogleMark, discord: DiscordMark };
  const Icon = icons[provider];
  return <Icon />;
};

interface InvitationProviderButtonProps {
  provider: InvitationProvider;
  disabled: boolean;
  onSelect: (provider: InvitationProvider) => void;
}

export const InvitationProviderButton = ({ provider, disabled, onSelect }: InvitationProviderButtonProps) => (
  <Button type="button" variant={provider === "github" ? "default" : "outline"} className="w-full" disabled={disabled} onClick={() => onSelect(provider)}>
    <ProviderIcon provider={provider} />Continue with {labels[provider]}
  </Button>
);

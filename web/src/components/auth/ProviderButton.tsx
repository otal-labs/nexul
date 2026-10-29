import { ProviderMark } from "@/components/ProviderMarks";
import { Button } from "@/components/ui/button";
import { providerLabel, type Provider } from "@/models/User";
import { cn } from "@/lib/utils";

interface ProviderButtonProps {
  provider: Provider;
  primary: boolean;
  disabled?: boolean;
  onSelect: (provider: Provider) => void;
}

// The mark sits at a fixed 16px from the edge, so a stack of providers lines its marks up whatever the label's length.
export const ProviderButton = ({ provider, primary, disabled = false, onSelect }: ProviderButtonProps) => (
  <Button
    type="button"
    size="lg"
    variant={primary ? "default" : "outline"}
    // The primary takes a clear border too, so its mark sits on the same x as the outline buttons' marks.
    className={cn("relative w-full [&>svg]:absolute [&>svg]:left-4", primary && "border border-transparent")}
    disabled={disabled}
    onClick={() => onSelect(provider)}
  >
    <ProviderMark provider={provider} />
    Continue with {providerLabel[provider]}
  </Button>
);

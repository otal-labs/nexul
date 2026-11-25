import { Building2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { joinAPIURL } from "@/api/client";
import type { SsoPublic } from "@/ee/models/Sso";
import { cn } from "@/lib/utils";

interface SsoSignInButtonProps {
  name: string;
  protocol: SsoPublic["protocol"];
  primary: boolean;
}

// The mark sits at a fixed 16px from the edge like the provider buttons', so a stack lines its marks up.
export const SsoSignInButton = ({ name, protocol, primary }: SsoSignInButtonProps) => (
  <Button
    type="button"
    size="lg"
    variant={primary ? "default" : "outline"}
    className={cn("relative w-full [&>svg]:absolute [&>svg]:left-4", primary && "border border-transparent")}
    onClick={() => window.location.assign(joinAPIURL(protocol === "saml" ? "/auth/saml" : "/auth/oidc"))}
  >
    <Building2 aria-hidden />
    Sign in with {name}
  </Button>
);

import type { ComponentType } from "react";
import { Unlink } from "lucide-react";

import { ConfirmDestroyButton } from "@/components/settings/ConfirmDestroyButton";
import { Button } from "@/components/ui/button";

interface IdentityRowProps {
  mark: ComponentType;
  provider: string;
  account?: string | undefined;
  onlyOne?: boolean;
}

export const IdentityRow = ({ mark: Mark, provider, account, onlyOne = false }: IdentityRowProps) => (
  <li className="flex items-center gap-3 bg-card px-3 py-3">
    <span className="flex size-9 shrink-0 items-center justify-center rounded-md border border-border bg-muted/60 [&_svg]:size-4">
      <Mark />
    </span>
    <div className="min-w-0 flex-1">
      <p className="text-sm font-medium">{provider}</p>
      <p className="truncate font-mono text-xs text-muted-foreground">{account ?? "Not linked"}</p>
    </div>
    {account && !onlyOne && <ConfirmDestroyButton icon={Unlink} idleLabel={`Unlink ${provider}`} confirmLabel="Unlink" onConfirm={() => {}} />}
    {account && onlyOne && <span className="text-xs text-muted-foreground">Your only sign-in</span>}
    {!account && (
      <Button variant="outline" size="sm">
        Link {provider}
      </Button>
    )}
  </li>
);

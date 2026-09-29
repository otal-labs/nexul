import { ArrowLeft } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";

interface WizardFooterProps {
  onBack?: (() => void) | undefined;
  // A ghost link that sits immediately left of the primary action.
  skip?: ReactNode;
  children?: ReactNode;
}

// The action row every step ends with: Back on the left, Skip then the primary action on the right.
export const WizardFooter = ({ onBack, skip, children }: WizardFooterProps) => {
  if (!onBack && !skip && !children) return null;
  return (
    <div className="mt-8 flex flex-wrap items-center gap-2">
      {onBack && (
        <Button type="button" variant="ghost" className="-ml-2 text-muted-foreground" onClick={onBack}>
          <ArrowLeft className="size-4" aria-hidden />
          Back
        </Button>
      )}
      <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
        {skip}
        {children}
      </div>
    </div>
  );
};

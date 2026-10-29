import { ArrowLeft } from "lucide-react";
import type { ReactNode } from "react";
import { createPortal } from "react-dom";

import { Button } from "@/components/ui/button";
import { useWizardBar } from "@/hooks/useWizardBar";

interface WizardFooterProps {
  onBack?: (() => void) | undefined;
  // A ghost link that sits immediately left of the primary action.
  skip?: ReactNode;
  children?: ReactNode;
}

// The action row every step ends with: Back on the left, Skip then the primary action on the right, in the sheet's bar when there is one.
export const WizardFooter = ({ onBack, skip, children }: WizardFooterProps) => {
  const bar = useWizardBar();
  if (!onBack && !skip && !children) return null;
  const row = (
    <div className="flex w-full items-center gap-2">
      {onBack && (
        <Button type="button" variant="ghost" className="-ml-3 text-muted-foreground" onClick={onBack}>
          <ArrowLeft className="size-4" aria-hidden />
          Back
        </Button>
      )}
      <div className="ml-auto flex items-center justify-end gap-2">
        {skip}
        {children}
      </div>
    </div>
  );
  return (
    <>
      {bar && createPortal(row, bar)}
      {!bar && <div className="mt-8">{row}</div>}
    </>
  );
};

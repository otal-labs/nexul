import type { ReactNode } from "react";
import { createPortal } from "react-dom";

import { useWizardBar } from "@/hooks/useWizardBar";

// A step's primary that other flows also render: in a wizard it moves to the pinned bar's right end, elsewhere it stays where it is.
export const InWizardBar = ({ children }: { children: ReactNode }) => {
  const bar = useWizardBar();
  return (
    <>
      {bar && createPortal(<div className="ml-auto flex items-center gap-2">{children}</div>, bar)}
      {!bar && children}
    </>
  );
};

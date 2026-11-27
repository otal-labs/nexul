import type { ReactNode } from "react";

// What a row unfolds into: on `surface-2`, the fields as label rows, inside a container so they stack in a narrow pane.
export const UnfoldedEditor = ({ label, children }: { label: string; children: ReactNode }) => (
  <div role="group" aria-label={label} className="settle-in @container border-b border-border bg-surface-2 px-4 pb-4 sm:px-6">
    {children}
  </div>
);

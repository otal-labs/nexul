import type { ReactNode } from "react";

interface EditorRowProps {
  label: string;
  children: ReactNode;
}

// One label-column row of an unfolded editor: the label at the left, its controls beside it (stacked in a narrow pane).
export const EditorRow = ({ label, children }: EditorRowProps) => (
  <section aria-label={label} className="grid gap-x-6 gap-y-2 border-t border-border py-4 first:border-t-0 @xl:grid-cols-[8rem_minmax(0,1fr)]">
    <h3 className="text-sm font-medium @xl:pt-1.5">{label}</h3>
    <div className="min-w-0 space-y-2">{children}</div>
  </section>
);

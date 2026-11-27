import type { ReactNode } from "react";

// Save and Cancel at an unfolded row's foot, the row's own ghost actions (Delete, History) at the left; it stays at the
// bottom of the screen while the panel is taller than it.
export const EditorFooter = ({ children }: { children: ReactNode }) => (
  <div className="sticky bottom-0 z-10 -mx-4 flex items-center gap-2 rounded-b-lg border-t border-border bg-surface-2 px-4 py-3 sm:-mx-6 sm:px-6">{children}</div>
);

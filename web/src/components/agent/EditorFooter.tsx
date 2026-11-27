import type { ReactNode } from "react";

// Save and Cancel at an unfolded row's foot, the row's own ghost actions (Delete, History) at the left.
export const EditorFooter = ({ children }: { children: ReactNode }) => <div className="flex items-center gap-2 border-t border-border pt-4">{children}</div>;

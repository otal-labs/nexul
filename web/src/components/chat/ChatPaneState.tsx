import type { ReactNode } from "react";

// Loading, error, and empty all sit centred in the space the messages would fill.
export const ChatPaneState = ({ children }: { children: ReactNode }) => (
  <div className="flex min-h-0 flex-1 items-center justify-center p-6">{children}</div>
);

import type { ReactNode } from "react";

import { PageHeader } from "@/components/PageHeader";
import { useWorkspaceCrumb } from "@/hooks/useCrumbs";

// Filling the page, the inbox opens like every other page; beside an open row it keeps the list pane's title bar.
export const InboxPageHeader = ({ unread, controls }: { unread: number; controls: ReactNode }) => (
  <PageHeader
    crumbs={[useWorkspaceCrumb()]}
    title="Inbox"
    meta={
      <span>
        {unread > 0 && (
          <>
            <span className="font-mono tabular-nums">{unread}</span> unread
          </>
        )}
        {unread === 0 && "Nothing unread"}
      </span>
    }
    actions={<div className="flex items-center gap-1.5">{controls}</div>}
    className="shrink-0 overflow-hidden px-3 pt-8 [scrollbar-gutter:stable_both-edges]"
  />
);

import { lazy, Suspense } from "react";

import { LoadingDisplay } from "@/components/LoadingDisplay";
import { NoDataDisplay } from "@/components/NoDataDisplay";
import { SubjectType } from "@/models/Notification";
import type { Notification } from "@/models/Notification";
import { TicketPage } from "@/pages/TicketPage";

// Lazy: DocPage pulls in @tiptap/* and yjs, which would otherwise bloat InboxPage's eager chunk.
const DocPage = lazy(() => import("@/pages/DocPage").then((m) => ({ default: m.DocPage })));
// MemoryPage shares DocPage's tiptap editor, so it stays out of the eager chunk for the same reason.
const MemoryPage = lazy(() => import("@/pages/MemoryPage").then((m) => ({ default: m.MemoryPage })));

interface NotificationDetailPanelProps {
  selected: Notification | null;
  isLoading: boolean;
}

export const NotificationDetailPanel = ({ selected, isLoading }: NotificationDetailPanelProps) => (
  <div className="min-w-0 flex-1 overflow-y-auto">
    {selected && selected.subject_type === SubjectType.Ticket && <TicketPage ticketId={selected.subject_id} />}
    {selected && selected.subject_type === SubjectType.Doc && (
      <Suspense fallback={<LoadingDisplay />}>
        <DocPage docId={selected.subject_id} />
      </Suspense>
    )}
    {selected && selected.subject_type === SubjectType.Memory && (
      <Suspense fallback={<LoadingDisplay />}>
        <MemoryPage memoryId={selected.subject_id} />
      </Suspense>
    )}
    {!selected && !isLoading && (
      <div className="flex h-full items-center justify-center">
        <NoDataDisplay message="Select a notification to view it" />
      </div>
    )}
  </div>
);

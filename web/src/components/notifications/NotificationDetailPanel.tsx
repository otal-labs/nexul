import { lazy, Suspense } from "react";

import { LoadingDisplay } from "@/components/LoadingDisplay";
import { SubjectType } from "@/models/Notification";
import type { Notification } from "@/models/Notification";
import { TicketPage } from "@/pages/TicketPage";

// Lazy: DocPage pulls in @tiptap/* and yjs, which would otherwise bloat InboxPage's eager chunk.
const DocPage = lazy(() => import("@/pages/DocPage").then((m) => ({ default: m.DocPage })));
// MemoryPage shares DocPage's tiptap editor, so it stays out of the eager chunk for the same reason.
const MemoryPage = lazy(() => import("@/pages/MemoryPage").then((m) => ({ default: m.MemoryPage })));

interface NotificationDetailPanelProps {
  selected: Notification | undefined;
}

export const NotificationDetailPanel = ({ selected }: NotificationDetailPanelProps) => (
  <div>
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
  </div>
);

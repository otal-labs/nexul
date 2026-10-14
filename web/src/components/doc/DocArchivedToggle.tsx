import { ArchiveIcon } from "lucide-react";

import { Toggle } from "@/components/ui/toggle";
import { useDocArchiveStore } from "@/stores/docArchiveStore";

// The list shows live docs; this swaps it to the archived ones, where each row offers Restore.
export const DocArchivedToggle = () => {
  const showArchived = useDocArchiveStore((s) => s.showArchived);
  const setShowArchived = useDocArchiveStore((s) => s.setShowArchived);
  return (
    <Toggle
      size="sm"
      className="size-8 min-w-8 px-0"
      pressed={showArchived}
      onPressedChange={setShowArchived}
      aria-label="Show archived docs"
      title={showArchived ? "Showing archived docs" : "Show archived docs"}
    >
      <ArchiveIcon className="size-4" aria-hidden />
    </Toggle>
  );
};

import { DangerZone } from "@/components/settings/DangerZone";
import { EmptyRow } from "@/components/EmptyRow";

// No workspace-wide action is destructive yet; a project is removed from its own settings.
export const DangerZoneSection = () => (
  <DangerZone>
    <li className="py-3">
      <EmptyRow flush>Nothing here can't be undone yet. To remove a project, open its settings.</EmptyRow>
    </li>
  </DangerZone>
);

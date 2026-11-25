import { arrayMove } from "@dnd-kit/sortable";

import type { RowAction } from "@/components/settings/RowActionsMenu";
import { useListComputers, useReorderComputers } from "@/hooks/PairingHooks";

// Move up and Move down for one of your computers: the keyboard's way to change the run order the list drags.
export const useComputerMoves = (computerId: string): RowAction[] => {
  const { data: computers } = useListComputers();
  const { mutate: reorder } = useReorderComputers();
  const ids = computers?.map((c) => c.id) ?? [];
  const index = ids.indexOf(computerId);
  if (index < 0 || ids.length < 2) return [];

  return [
    { label: "Move up", disabled: index === 0, onSelect: () => reorder(arrayMove(ids, index, index - 1)) },
    { label: "Move down", disabled: index === ids.length - 1, onSelect: () => reorder(arrayMove(ids, index, index + 1)) },
  ];
};

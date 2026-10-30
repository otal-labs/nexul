import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useInboxFilterStore, type InboxFilter } from "@/stores/inboxFilterStore";

export const InboxFilterToggle = () => {
  const filter = useInboxFilterStore((s) => s.filter);
  const setFilter = useInboxFilterStore((s) => s.setFilter);
  return (
    <ToggleGroup
      type="single"
      variant="segmented"
      size="xs"
      value={filter}
      // Radix reports "" when the chosen side is pressed again; the filter stands.
      onValueChange={(value) => value && setFilter(value as InboxFilter)}
      aria-label="Show notifications"
    >
      <ToggleGroupItem value="unread">Unread</ToggleGroupItem>
      <ToggleGroupItem value="all">All</ToggleGroupItem>
    </ToggleGroup>
  );
};

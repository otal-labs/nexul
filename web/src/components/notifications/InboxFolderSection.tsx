import { FolderToggle } from "@/components/listpane/FolderToggle";
import { NotificationItem } from "@/components/notifications/NotificationItem";
import { useInboxFolderStore } from "@/stores/inboxFolderStore";
import type { InboxFolder } from "@/utils/InboxUtility";
import { cn } from "@/lib/utils";

interface InboxFolderSectionProps {
  folder: InboxFolder;
}

export const InboxFolderSection = ({ folder }: InboxFolderSectionProps) => {
  const open = !useInboxFolderStore((s) => s.collapsed.includes(folder.key));
  const toggle = useInboxFolderStore((s) => s.toggleCollapsed);
  return (
    <section aria-label={folder.name}>
      <div className="flex h-9 items-center gap-2.5 border-b border-border px-4">
        <span className={cn("size-2 shrink-0 rounded-full", folder.unread && "bg-primary")} aria-label={folder.unread ? "Unread" : undefined} />
        <FolderToggle
          name={folder.name}
          open={open}
          onToggle={() => toggle(folder.key)}
          meta={folder.summary}
          metaClassName="min-w-0 shrink grow basis-0 truncate text-right normal-case tracking-normal"
          className={cn(folder.unread && "text-foreground")}
        />
      </div>
      {open && folder.rows.map((row) => <NotificationItem key={row.key} row={row} inset />)}
    </section>
  );
};

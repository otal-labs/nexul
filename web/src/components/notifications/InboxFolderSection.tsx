import { FolderToggle } from "@/components/listpane/FolderToggle";
import { NotificationItem } from "@/components/notifications/NotificationItem";
import { useInboxFolderStore } from "@/stores/inboxFolderStore";
import type { InboxFolder } from "@/utils/InboxUtility";
import { cn } from "@/lib/utils";

interface InboxFolderSectionProps {
  folder: InboxFolder;
}

// Open, the folder's own rows carry their unread dots; collapsed, the folder carries one for them.
export const InboxFolderSection = ({ folder }: InboxFolderSectionProps) => {
  const open = !useInboxFolderStore((s) => s.collapsed.includes(folder.key));
  const toggle = useInboxFolderStore((s) => s.toggleCollapsed);
  const showDot = folder.unread && !open;
  return (
    <section aria-label={folder.name}>
      <div className="flex h-9 items-center px-3 has-[:focus-visible]:outline-2 has-[:focus-visible]:-outline-offset-2 has-[:focus-visible]:rounded-md has-[:focus-visible]:outline-(--focus)">
        <span className="flex w-3 shrink-0 items-center">
          <span
            {...(showDot ? { role: "img", "aria-label": "Unread" } : { "aria-hidden": true })}
            data-unread={showDot || undefined}
            className="unread-dot size-1.5 rounded-full bg-foreground"
          />
        </span>
        <FolderToggle
          name={folder.name}
          open={open}
          onToggle={() => toggle(folder.key)}
          meta={folder.rows.length}
          metaClassName="text-muted-foreground"
          className={cn("quiet-focus", folder.unread && "text-foreground")}
        />
      </div>
      {open && folder.rows.map((row) => <NotificationItem key={row.key} row={row} />)}
    </section>
  );
};

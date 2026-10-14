import {
  ArchiveIcon,
  ArchiveRestoreIcon,
  BotIcon,
  CopyIcon,
  EyeIcon,
  EyeOffIcon,
  FilePlusIcon,
  FolderInputIcon,
  LockIcon,
  LockOpenIcon,
  MailIcon,
  PencilIcon,
  PinIcon,
  PinOffIcon,
  Settings2Icon,
  ShieldIcon,
  Trash2Icon,
} from "lucide-react";

import { ActionMenuItem } from "@/components/action/ActionMenuItem";
import { hasOwnActions, type MenuParts, type RowMenu } from "@/components/listpane/rowMenu";
import { useCanManageAccess } from "@/hooks/useCanManageAccess";

interface RowActionItemsProps extends RowMenu {
  ui: MenuParts;
}

// One set of items for the ⋯ dropdown and the right-click menu: the row's actions, Permissions…, then Delete.
export const RowActionItems = ({ ui, ...m }: RowActionItemsProps) => {
  const own = hasOwnActions(m);
  return (
    <>
      {m.onSettings && (
        <ui.Item onSelect={m.onSettings}>
          <Settings2Icon aria-hidden />
          {m.settingsLabel ?? "Settings"}
        </ui.Item>
      )}
      {m.onBots && (
        <ui.Item onSelect={m.onBots}>
          <BotIcon aria-hidden />
          Bots
        </ui.Item>
      )}
      {m.onNewDoc && (
        <ui.Item onSelect={m.onNewDoc}>
          <FilePlusIcon aria-hidden />
          New doc
        </ui.Item>
      )}
      {m.pin && (
        <ui.Item onSelect={m.pin.onToggle}>
          {m.pin.pinned && <PinOffIcon aria-hidden />}
          {!m.pin.pinned && <PinIcon aria-hidden />}
          {m.pin.pinned ? "Unpin" : "Pin"}
        </ui.Item>
      )}
      {m.sidebar && !m.sidebar.hidden && (
        <ui.Item onSelect={m.sidebar.onToggle}>
          <EyeOffIcon aria-hidden />
          Remove from sidebar
        </ui.Item>
      )}
      {m.sidebar?.hidden && (
        <ui.Item onSelect={m.sidebar.onToggle}>
          <EyeIcon aria-hidden />
          Show in sidebar
        </ui.Item>
      )}
      {m.lock && !m.lock.locked && (
        <ui.Item onSelect={m.lock.onToggle}>
          <LockIcon aria-hidden />
          Lock
        </ui.Item>
      )}
      {m.lock?.locked && (
        <ui.Item onSelect={m.lock.onToggle}>
          <LockOpenIcon aria-hidden />
          Unlock
        </ui.Item>
      )}
      {m.archive && !m.archive.archived && (
        <ui.Item onSelect={m.archive.onToggle}>
          <ArchiveIcon aria-hidden />
          Archive
        </ui.Item>
      )}
      {m.archive?.archived && (
        <ui.Item onSelect={m.archive.onToggle}>
          <ArchiveRestoreIcon aria-hidden />
          Restore
        </ui.Item>
      )}
      {m.moveTo && (
        <ui.Sub>
          <ui.SubTrigger>
            <FolderInputIcon aria-hidden />
            {m.moveTo.label}
          </ui.SubTrigger>
          <ui.SubContent className="min-w-36">
            <ui.RadioGroup value={m.moveTo.currentId} onValueChange={m.moveTo.onMove}>
              {m.moveTo.options.map((option) => (
                <ui.RadioItem key={option.id} value={option.id}>
                  {option.name}
                </ui.RadioItem>
              ))}
            </ui.RadioGroup>
          </ui.SubContent>
        </ui.Sub>
      )}
      {m.onMarkUnread && (
        <ui.Item onSelect={m.onMarkUnread}>
          <MailIcon aria-hidden />
          Mark unread
        </ui.Item>
      )}
      {m.onEdit && (
        <ui.Item onSelect={m.onEdit}>
          <PencilIcon aria-hidden />
          Edit
        </ui.Item>
      )}
      {m.onRename && (
        <ui.Item onSelect={m.onRename}>
          <PencilIcon aria-hidden />
          Rename
        </ui.Item>
      )}
      {m.onClone && (
        <ui.Item onSelect={m.onClone}>
          <CopyIcon aria-hidden />
          Clone
        </ui.Item>
      )}
      {m.permissions && <PermissionsItem ui={ui} permissions={m.permissions} afterOwn={own} beforeDelete={!!m.onDelete} />}
      {!m.permissions && m.onDelete && own && <ui.Separator />}
      {m.onDelete && !m.deleteGate && (
        <ui.Item variant="destructive" onSelect={m.onDelete}>
          <Trash2Icon aria-hidden />
          Delete
        </ui.Item>
      )}
      {m.onDelete && m.deleteGate && (
        <ActionMenuItem {...m.deleteGate} ui={ui} icon={Trash2Icon} destructive onSelect={m.onDelete}>
          Delete
        </ActionMenuItem>
      )}
    </>
  );
};

interface PermissionsItemProps {
  ui: MenuParts;
  permissions: NonNullable<RowMenu["permissions"]>;
  afterOwn: boolean;
  beforeDelete: boolean;
}

// Its own component so only a row offering Permissions… runs the check, and the rules around it follow the answer.
const PermissionsItem = ({ ui, permissions, afterOwn, beforeDelete }: PermissionsItemProps) => {
  const canManage = useCanManageAccess(permissions.entity);
  return (
    <>
      {canManage && afterOwn && <ui.Separator />}
      {canManage && (
        <ui.Item onSelect={permissions.onOpen}>
          <ShieldIcon aria-hidden />
          Permissions…
        </ui.Item>
      )}
      {beforeDelete && (afterOwn || canManage) && <ui.Separator />}
    </>
  );
};

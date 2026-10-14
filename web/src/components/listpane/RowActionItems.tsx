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
  type LucideIcon,
} from "lucide-react";

import type { ReactNode } from "react";

import type { ActionGate } from "@nexul/client-core/actionState";

import { ActionMenuItem } from "@/components/action/ActionMenuItem";
import { hasOwnActions, type MenuParts, type RowMenu } from "@/components/listpane/rowMenu";
import { useActionState } from "@/hooks/useActionState";
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
        <Entry ui={ui} gate={m.gates?.onNewDoc} icon={FilePlusIcon} onSelect={m.onNewDoc}>
          New doc
        </Entry>
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
        <Entry ui={ui} gate={m.gates?.lock} icon={LockIcon} onSelect={m.lock.onToggle}>
          Lock
        </Entry>
      )}
      {m.lock?.locked && (
        <Entry ui={ui} gate={m.gates?.lock} icon={LockOpenIcon} onSelect={m.lock.onToggle}>
          Unlock
        </Entry>
      )}
      {m.archive && !m.archive.archived && (
        <Entry ui={ui} gate={m.gates?.archive} icon={ArchiveIcon} onSelect={m.archive.onToggle}>
          Archive
        </Entry>
      )}
      {m.archive?.archived && (
        <Entry ui={ui} gate={m.gates?.archive} icon={ArchiveRestoreIcon} onSelect={m.archive.onToggle}>
          Restore
        </Entry>
      )}
      {m.moveTo && <MoveToItem ui={ui} gate={m.gates?.moveTo} moveTo={m.moveTo} />}
      {m.onMarkUnread && (
        <ui.Item onSelect={m.onMarkUnread}>
          <MailIcon aria-hidden />
          Mark unread
        </ui.Item>
      )}
      {m.onEdit && (
        <Entry ui={ui} gate={m.gates?.onEdit} icon={PencilIcon} onSelect={m.onEdit}>
          Edit
        </Entry>
      )}
      {m.onRename && (
        <Entry ui={ui} gate={m.gates?.onRename} icon={PencilIcon} onSelect={m.onRename}>
          Rename
        </Entry>
      )}
      {m.onClone && (
        <Entry ui={ui} gate={m.gates?.onClone} icon={CopyIcon} onSelect={m.onClone}>
          Clone
        </Entry>
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

interface EntryProps {
  ui: MenuParts;
  gate: ActionGate | undefined;
  icon: LucideIcon;
  onSelect: () => void;
  children: ReactNode;
}

// An item that changes something draws through the action state when its row brings a gate, else as a plain item.
const Entry = ({ ui, gate, icon: Icon, onSelect, children }: EntryProps) => (
  <>
    {gate && (
      <ActionMenuItem {...gate} ui={ui} icon={Icon} onSelect={onSelect}>
        {children}
      </ActionMenuItem>
    )}
    {!gate && (
      <ui.Item onSelect={onSelect}>
        <Icon aria-hidden />
        {children}
      </ui.Item>
    )}
  </>
);

interface MoveToItemProps {
  ui: MenuParts;
  gate: ActionGate | undefined;
  moveTo: NonNullable<RowMenu["moveTo"]>;
}

const MoveToItem = ({ ui, gate, moveTo }: MoveToItemProps) => (
  <>
    {gate && <GatedMoveTo ui={ui} gate={gate} moveTo={moveTo} />}
    {!gate && <MoveToSub ui={ui} moveTo={moveTo} />}
  </>
);

// Locked, the submenu is a single locked item; otherwise the places to move to.
const GatedMoveTo = ({ ui, gate, moveTo }: MoveToItemProps & { gate: ActionGate }) => {
  const action = useActionState(gate);
  return (
    <>
      {action.state === "locked" && (
        <ActionMenuItem {...gate} ui={ui} icon={FolderInputIcon} onSelect={() => undefined}>
          {moveTo.label}
        </ActionMenuItem>
      )}
      {action.state !== "locked" && <MoveToSub ui={ui} moveTo={moveTo} />}
    </>
  );
};

const MoveToSub = ({ ui, moveTo }: Omit<MoveToItemProps, "gate">) => (
  <ui.Sub>
    <ui.SubTrigger>
      <FolderInputIcon aria-hidden />
      {moveTo.label}
    </ui.SubTrigger>
    <ui.SubContent className="min-w-36">
      <ui.RadioGroup value={moveTo.currentId} onValueChange={moveTo.onMove}>
        {moveTo.options.map((option) => (
          <ui.RadioItem key={option.id} value={option.id}>
            {option.name}
          </ui.RadioItem>
        ))}
      </ui.RadioGroup>
    </ui.SubContent>
  </ui.Sub>
);

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

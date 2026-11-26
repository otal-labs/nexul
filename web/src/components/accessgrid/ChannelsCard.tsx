import type { ReactNode } from "react";
import { Hash, Volume2 } from "lucide-react";

import { AccessGridTable } from "@/components/accessgrid/AccessGridTable";
import { SeenByCell } from "@/components/accessgrid/SeenByCell";
import { EmptyRow } from "@/components/EmptyRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useAccessDraft } from "@/hooks/useAccessDraft";
import type { AccessGrid } from "@/models/AccessGrid";

interface ChannelsCardProps {
  grid: AccessGrid;
  footer: ReactNode;
  aside?: ReactNode;
  fill?: boolean;
}

// The same grid the other way round: every member sees a public channel, so a cell clears to hide it from a role, and Seen by reads the exceptions back.
export const ChannelsCard = ({ grid, footer, aside, fill = false }: ChannelsCardProps) => {
  const draft = useAccessDraft(grid);
  const rows = grid.channels.map((channel) => {
    const hidden = draft.hidden(channel.id);
    return {
      id: channel.id,
      name: channel.name,
      mark: (
        <span className="grid size-6 shrink-0 place-items-center rounded-md bg-surface-2 ring-1 ring-border" aria-hidden>
          {channel.kind === "voice_channel" && <Volume2 className="size-3.5 text-muted-foreground" />}
          {channel.kind !== "voice_channel" && <Hash className="size-3.5 text-muted-foreground" />}
        </span>
      ),
      trailing: <SeenByCell hiddenFrom={grid.roles.filter((role) => hidden.includes(role.id)).map((role) => role.name)} />,
    };
  });

  return (
    <SettingsCard
      id="channels"
      title="Public channels"
      description="Every member sees a public channel. Click a role's cell to hide a channel from it. Private channels keep their own members."
      aside={aside}
      footer={footer}
      fill={fill}
    >
      {rows.length === 0 && <EmptyRow flush>No public channels.</EmptyRow>}
      {rows.length > 0 && (
        <AccessGridTable
          label="Channel"
          leadLabel="Every channel"
          rows={rows}
          roles={grid.roles}
          marks="hidden"
          isOn={(rowId, roleId) => !draft.hidden(rowId).includes(roleId)}
          isChanged={(rowId, roleId) => draft.hiddenChanged(rowId, roleId)}
          onToggle={draft.toggleHidden}
          onSetAll={(roleId, visible) => draft.setRoleOnChannels(roleId, rows.map((row) => row.id), visible)}
          trailingHead="Seen by"
          fill={fill}
        />
      )}
    </SettingsCard>
  );
};

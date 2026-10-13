import type { ReactNode } from "react";
import { Hash, Volume2 } from "lucide-react";

import { AccessGridTable } from "@/components/accessgrid/AccessGridTable";
import { EmptyRow } from "@/components/EmptyRow";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { useAccessDraft } from "@/hooks/useAccessDraft";
import { countLabel, type AccessGrid } from "@/models/AccessGrid";

interface ChannelsCardProps {
  grid: AccessGrid;
  footer: ReactNode;
  aside?: ReactNode;
  // Under the projects grid on one page, an empty trailing column keeps the role columns in line with it.
  alignWithProjects?: boolean;
}

// The same grid the other way round: every box starts ticked, because every member sees a public channel.
export const ChannelsCard = ({ grid, footer, aside, alignWithProjects = false }: ChannelsCardProps) => {
  const draft = useAccessDraft(grid);
  const rows = grid.channels.map((channel) => {
    const hidden = draft.hidden(channel.id).length;
    return {
      id: channel.id,
      name: channel.name,
      mark: (
        <span className="grid size-7 shrink-0 place-items-center rounded-md bg-surface-2 ring-1 ring-border" aria-hidden>
          {channel.kind === "voice_channel" && <Volume2 className="size-3.5 text-muted-foreground" />}
          {channel.kind !== "voice_channel" && <Hash className="size-3.5 text-muted-foreground" />}
        </span>
      ),
      meta: hidden > 0 ? `Hidden from ${countLabel(hidden, "role", "roles")}` : undefined,
    };
  });

  return (
    <SettingsCard
      id="channels"
      title="Public channels"
      description="Every member sees a public channel. Untick a role to hide a channel from it. Private channels keep their own members."
      aside={aside}
      footer={footer}
    >
      {rows.length === 0 && <EmptyRow flush>No public channels.</EmptyRow>}
      {rows.length > 0 && (
        <AccessGridTable
          label="Channel"
          leadLabel="Every channel"
          rows={rows}
          roles={grid.roles}
          isOn={(rowId, roleId) => !draft.hidden(rowId).includes(roleId)}
          onToggle={draft.toggleHidden}
          onSetAll={(roleId, visible) => draft.setRoleOnChannels(roleId, rows.map((row) => row.id), visible)}
          trailingHead={alignWithProjects ? "" : undefined}
        />
      )}
    </SettingsCard>
  );
};

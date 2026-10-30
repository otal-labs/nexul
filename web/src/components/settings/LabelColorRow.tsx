import { ColorSwatchButton } from "@/components/settings/ColorSwatchButton";
import { SettingsListRow } from "@/components/settings/SettingsListRow";
import { HUE_DOT_CLASS } from "@/components/board/ticketTypeColor";
import { useFetchProjectTicketCounts } from "@/hooks/ProjectHooks";
import { useSetLabelColor } from "@/hooks/TicketHooks";
import { countLabel } from "@/utils/CountUtility";

interface LabelColorRowProps {
  label: string;
  /** "" when the label has no configured color yet (the board falls back to a hashed hue). */
  color: string;
  projectId: string;
}

// Labels aren't a real entity (ADR 0005): the pill is the one action, its color; an unset one shows the hollow ring, not the board's fallback.
export const LabelColorRow = ({ label, color, projectId }: LabelColorRowProps) => {
  const setLabelColor = useSetLabelColor();
  const { data: counts } = useFetchProjectTicketCounts(projectId);
  const unset = !HUE_DOT_CLASS[color];

  return (
    <SettingsListRow
      name={
        <ColorSwatchButton
          permission="tickets:write"
          verb="recolor labels"
          projectId={projectId}
          label={`Color for label ${label}`}
          value={color}
          pill={label}
          allowNone={false}
          onChange={(next) => void setLabelColor.mutateAsync({ label, color: next, project_id: projectId })}
        />
      }
      meta={unset && "No color"}
      count={counts && countLabel(counts.tickets_by_label[label] ?? 0, "ticket")}
      // No menu, but the `⋯` column is kept so the counts line up with every other list's.
      actions={<span aria-hidden className="w-8" />}
      showActions
    />
  );
};

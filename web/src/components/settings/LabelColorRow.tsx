import { ColorSwatchButton } from "@/components/settings/ColorSwatchButton";
import { RowActionsMenu } from "@/components/settings/RowActionsMenu";
import { SettingsListRow } from "@/components/settings/SettingsListRow";
import { HUE_DOT_CLASS } from "@/components/board/ticketTypeColor";
import { useFetchProjectTicketCounts } from "@/hooks/ProjectHooks";
import { useDeleteLabel, useSetLabelColor } from "@/hooks/LabelHooks";
import { countLabel } from "@/utils/CountUtility";

interface LabelColorRowProps {
  label: string;
  /** "" when the label has no configured color yet (the board falls back to a hashed hue). */
  color: string;
  projectId: string;
}

// The pill sets the color, and an unset one shows the hollow ring, not the board's fallback; only an unused label can be deleted.
export const LabelColorRow = ({ label, color, projectId }: LabelColorRowProps) => {
  const setLabelColor = useSetLabelColor();
  const deleteLabel = useDeleteLabel();
  const { data: counts } = useFetchProjectTicketCounts(projectId);
  const unset = !HUE_DOT_CLASS[color];
  const used = counts?.tickets_by_label[label] ?? 0;

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
      count={counts && countLabel(used, "ticket")}
      actions={
        <RowActionsMenu
          subject={label}
          projectId={projectId}
          actions={[
            {
              label: "Delete",
              destructive: true,
              permission: "tickets:write",
              verb: "delete labels",
              disabled: used > 0,
              hint: used > 0 ? `On ${countLabel(used, "ticket")}` : undefined,
              onSelect: () => deleteLabel.mutate({ label, project_id: projectId }),
            },
          ]}
        />
      }
    />
  );
};

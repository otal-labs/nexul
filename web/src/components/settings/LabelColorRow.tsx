import { ColorPicker } from "@/components/settings/ColorPicker";
import { useSetLabelColor } from "@/hooks/TicketHooks";

interface LabelColorRowProps {
  label: string;
  /** "" when the label has no configured color yet (board falls back to the hash). */
  color: string;
  projectId: string;
}

// Labels aren't a real entity (ADR 0005) — color only, no rename/delete/unset.
export const LabelColorRow = ({ label, color, projectId }: LabelColorRowProps) => {
  const setLabelColor = useSetLabelColor();

  return (
    <li className="flex items-center gap-2 px-3 py-1 text-sm">
      <span className="min-w-0 flex-1 truncate text-sm font-medium" title={label}>{label}</span>
      <ColorPicker
        label={`Color for label ${label}`}
        value={color}
        allowNone={false}
        onChange={(next) => void setLabelColor.mutateAsync({ label, color: next, project_id: projectId })}
      />
    </li>
  );
};

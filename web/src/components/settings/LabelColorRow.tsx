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
    <li className="-mx-2 flex items-center gap-2 rounded-sm px-2 py-1.5 text-sm transition-colors duration-[120ms] ease-standard hover:bg-accent/40">
      <span className="flex-1 text-sm font-medium">{label}</span>
      <ColorPicker
        label={`Color for label ${label}`}
        value={color}
        allowNone={false}
        onChange={(next) => void setLabelColor.mutateAsync({ label, color: next, project_id: projectId })}
      />
    </li>
  );
};

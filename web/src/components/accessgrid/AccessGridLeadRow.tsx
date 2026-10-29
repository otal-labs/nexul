import { AccessGridLeadCell } from "@/components/accessgrid/AccessGridLeadCell";
import { fadeEndCell, leadStrip, pinnedStrip, stickyCell, stickyStrip } from "@/components/accessgrid/gridStyles";
import type { AccessGridRole } from "@/models/AccessGrid";
import { cn } from "@/lib/utils";

interface AccessGridLeadRowProps {
  label: string;
  roles: AccessGridRole[];
  rowIds: string[];
  marks: "ticks" | "hidden";
  isOn: (rowId: string, roleId: string) => boolean;
  onSetAll: (roleId: string, on: boolean) => void;
  trailing: boolean;
}

// "Every project": one cell per role that sets it on every row shown, with how many rows hold it.
export const AccessGridLeadRow = ({ label, roles, rowIds, marks, isOn, onSetAll, trailing }: AccessGridLeadRowProps) => (
  <tr>
    <th scope="row" className={cn(stickyCell, stickyStrip, "h-11 border-b border-border pr-3 pb-1 pl-6 text-left text-sm font-medium")}>
      {label}
    </th>
    {roles.map((role) => (
      <AccessGridLeadCell
        key={role.id}
        on={rowIds.filter((id) => isOn(id, role.id)).length}
        rows={rowIds.length}
        marks={marks}
        label={`${role.name}: ${label.toLowerCase()}`}
        onSetAll={(on) => onSetAll(role.id, on)}
      />
    ))}
    <td aria-hidden className={leadStrip} />
    {trailing && <td className={pinnedStrip} />}
    {!trailing && <td aria-hidden className={cn(fadeEndCell, "bg-card", stickyStrip)} />}
  </tr>
);

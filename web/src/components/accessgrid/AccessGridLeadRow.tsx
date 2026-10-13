import { LeadCheckbox } from "@/components/accessgrid/LeadCheckbox";
import { leadStrip, stickyCell, stickyStrip } from "@/components/accessgrid/gridStyles";
import { leadState, type AccessGridRole } from "@/models/AccessGrid";
import { cn } from "@/lib/utils";

interface AccessGridLeadRowProps {
  label: string;
  roles: AccessGridRole[];
  rowIds: string[];
  isOn: (rowId: string, roleId: string) => boolean;
  onSetAll: (roleId: string, on: boolean) => void;
  trailing: boolean;
}

// "Every project": one box per role that ticks it on every row shown, a dash while only some rows have it.
export const AccessGridLeadRow = ({ label, roles, rowIds, isOn, onSetAll, trailing }: AccessGridLeadRowProps) => (
  <tr>
    <th scope="row" className={cn(stickyCell, stickyStrip, "border-b border-border py-2 pr-3 pl-6 text-left font-normal")}>
      <span className="block text-sm font-medium">{label}</span>
      <span className="block text-xs whitespace-nowrap text-muted-foreground">Tick a role everywhere</span>
    </th>
    {roles.map((role) => (
      <td key={role.id} className={cn(leadStrip, "px-1.5 text-center")}>
        <span className="inline-grid size-9 place-items-center">
          <LeadCheckbox
            checked={leadState(rowIds.filter((id) => isOn(id, role.id)).length, rowIds.length)}
            onCheckedChange={(on) => onSetAll(role.id, on)}
            label={`${role.name}: ${label.toLowerCase()}`}
          />
        </span>
      </td>
    ))}
    {trailing && <td className={leadStrip} />}
  </tr>
);

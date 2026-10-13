import { microheaderClass } from "@/components/Microheader";
import { AccessGridLeadRow } from "@/components/accessgrid/AccessGridLeadRow";
import { AccessGridRow, type AccessGridRowModel } from "@/components/accessgrid/AccessGridRow";
import { RoleColumnHead } from "@/components/accessgrid/RoleColumnHead";
import { stickyCell } from "@/components/accessgrid/gridStyles";
import type { AccessGridRole } from "@/models/AccessGrid";
import { cn } from "@/lib/utils";

interface AccessGridTableProps {
  label: string;
  leadLabel: string;
  rows: AccessGridRowModel[];
  roles: AccessGridRole[];
  isOn: (rowId: string, roleId: string) => boolean;
  onToggle: (rowId: string, roleId: string) => void;
  onSetAll: (roleId: string, on: boolean) => void;
  // A trailing column's head; "" keeps an empty one so two grids stacked on a page line their role columns up.
  trailingHead?: string | undefined;
}

// Past a dozen rows the grid scrolls inside the card, so the role heads stay in view over the thirtieth row.
const TALL_FROM = 12;

// Rows by roles, the name column sticky on the left and the head row sticky on top.
export const AccessGridTable = ({ label, leadLabel, rows, roles, isOn, onToggle, onSetAll, trailingHead }: AccessGridTableProps) => (
  <div className={cn("-mx-6 -mb-6 overflow-x-auto overscroll-contain", rows.length > TALL_FROM && "max-h-[min(60vh,40rem)] overflow-y-auto")}>
    <table className="w-full border-separate border-spacing-0 text-sm [&_tbody_tr:last-child>*]:border-b-0">
      <thead>
        <tr>
          <th scope="col" className={cn(stickyCell, microheaderClass, "top-0 z-30 min-w-40 border-b border-border pr-3 pb-2 pl-6 text-left align-bottom")}>
            {label}
          </th>
          {roles.map((role) => (
            <RoleColumnHead key={role.id} role={role} />
          ))}
          {trailingHead !== undefined && (
            <th scope="col" className={cn(microheaderClass, "sticky top-0 z-20 w-24 min-w-24 border-b border-border bg-card pr-4 pb-2 pl-3 text-left align-bottom")}>
              {trailingHead}
            </th>
          )}
        </tr>
      </thead>
      <tbody>
        <AccessGridLeadRow label={leadLabel} roles={roles} rowIds={rows.map((row) => row.id)} isOn={isOn} onSetAll={onSetAll} trailing={trailingHead !== undefined} />
        {rows.map((row) => (
          <AccessGridRow key={row.id} row={row} roles={roles} isOn={isOn} onToggle={onToggle} trailing={trailingHead !== undefined} />
        ))}
      </tbody>
    </table>
  </div>
);

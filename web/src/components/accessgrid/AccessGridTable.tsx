import { microheaderClass } from "@/components/Microheader";
import { AccessGridLeadRow } from "@/components/accessgrid/AccessGridLeadRow";
import { AccessGridRow, type AccessGridRowModel } from "@/components/accessgrid/AccessGridRow";
import { RoleColumnHead } from "@/components/accessgrid/RoleColumnHead";
import { fadeEndCell, fillerCell, snapScroller, stickyCell } from "@/components/accessgrid/gridStyles";
import { ScrollEndFade } from "@/components/ScrollEndFade";
import type { AccessGridRole } from "@/models/AccessGrid";
import { cn } from "@/lib/utils";

interface AccessGridTableProps {
  label: string;
  leadLabel: string;
  rows: AccessGridRowModel[];
  roles: AccessGridRole[];
  marks: "ticks" | "hidden";
  isOn: (rowId: string, roleId: string) => boolean;
  isChanged: (rowId: string, roleId: string) => boolean;
  onToggle: (rowId: string, roleId: string) => void;
  onSetAll: (roleId: string, on: boolean) => void;
  // A trailing column's head; "" keeps an empty one so two grids stacked on a page line their role columns up.
  trailingHead?: string | undefined;
  // Fills its card's height and scrolls both ways itself; otherwise a tall grid stops at 60vh.
  fill?: boolean;
}

const TALL_FROM = 12;

// Rows by roles in one scroller (paint-contained: without it its sticky cells widened the page at 768), its head row sticky on top and its name column sticky on the left.
export const AccessGridTable = ({ label, leadLabel, rows, roles, marks, isOn, isChanged, onToggle, onSetAll, trailingHead, fill = false }: AccessGridTableProps) => (
  <div
    className={cn(
      "-mx-6 -mb-6 min-h-0 overflow-auto overscroll-contain [contain:paint] [scrollbar-gutter:stable]",
      snapScroller,
      !fill && rows.length > TALL_FROM && "max-h-[min(60vh,40rem)]",
    )}
  >
    <table className="w-full border-separate border-spacing-0 text-sm [&_tbody_tr:last-child>*]:border-b-0">
      <thead>
        <tr>
          <th scope="col" className={cn(stickyCell, microheaderClass, "top-0 z-30 border-b border-border pr-3 pb-2 pl-6 text-left align-bottom")}>
            {label}
          </th>
          {roles.map((role) => (
            <RoleColumnHead key={role.id} role={role} />
          ))}
          <th aria-hidden className={cn(fillerCell, "sticky top-0 z-20 bg-card")} />
          {trailingHead !== undefined && (
            <th scope="col" className={cn(microheaderClass, "grid-edge-end sticky top-0 right-0 z-20 w-32 min-w-32 border-b border-border bg-card pr-6 pb-2 pl-3 text-left align-bottom")}>
              {trailingHead}
            </th>
          )}
          {trailingHead === undefined && <th aria-hidden className={cn(fadeEndCell, "top-0 z-20 bg-card")} />}
        </tr>
      </thead>
      <tbody>
        <AccessGridLeadRow label={leadLabel} roles={roles} rowIds={rows.map((row) => row.id)} marks={marks} isOn={isOn} onSetAll={onSetAll} trailing={trailingHead !== undefined} />
        {rows.map((row) => (
          <AccessGridRow key={row.id} row={row} roles={roles} marks={marks} isOn={isOn} isChanged={isChanged} onToggle={onToggle} trailing={trailingHead !== undefined} />
        ))}
      </tbody>
    </table>
    <ScrollEndFade />
  </div>
);

import type { ReactNode } from "react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { AccessGridCell } from "@/components/accessgrid/AccessGridCell";
import { fadeEndCell, fillerCell, nameText, stickyCell, stickyHover } from "@/components/accessgrid/gridStyles";
import type { AccessGridRole } from "@/models/AccessGrid";
import { cn } from "@/lib/utils";

export interface AccessGridRowModel {
  id: string;
  name: string;
  mark: ReactNode;
  trailing?: ReactNode;
}

interface AccessGridRowProps {
  row: AccessGridRowModel;
  roles: AccessGridRole[];
  marks: "ticks" | "hidden";
  isOn: (rowId: string, roleId: string) => boolean;
  isChanged: (rowId: string, roleId: string) => boolean;
  onToggle: (rowId: string, roleId: string) => void;
  trailing: boolean;
}

export const AccessGridRow = ({ row, roles, marks, isOn, isChanged, onToggle, trailing }: AccessGridRowProps) => (
  <tr className="group/row">
    <th scope="row" className={cn(stickyCell, stickyHover, "h-11 border-b border-border pr-3 pl-6 text-left font-normal")}>
      <span className="flex min-w-0 items-center gap-2.5">
        {row.mark}
        <span className="relative h-5 min-w-0 flex-1">
          <Tooltip>
            <TooltipTrigger asChild>
              <span className={nameText}>{row.name}</span>
            </TooltipTrigger>
            <TooltipContent>{row.name}</TooltipContent>
          </Tooltip>
        </span>
      </span>
    </th>
    {roles.map((role) => (
      <AccessGridCell
        key={role.id}
        on={isOn(row.id, role.id)}
        changed={isChanged(row.id, role.id)}
        marks={marks}
        label={`${role.name} in ${row.name}`}
        onToggle={() => onToggle(row.id, role.id)}
      />
    ))}
    <td aria-hidden className={cn(fillerCell, "group-hover/row:bg-accent/40")} />
    {trailing && <td className="grid-edge-end sticky right-0 z-10 border-b border-border bg-card py-1 pr-6 pl-3 group-hover/row:bg-linear-to-r group-hover/row:from-accent/40 group-hover/row:to-accent/40">{row.trailing}</td>}
    {!trailing && <td aria-hidden className={fadeEndCell} />}
  </tr>
);

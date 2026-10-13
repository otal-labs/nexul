import type { ReactNode } from "react";

import { Checkbox } from "@/components/ui/checkbox";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { stickyCell, stickyHover } from "@/components/accessgrid/gridStyles";
import type { AccessGridRole } from "@/models/AccessGrid";
import { cn } from "@/lib/utils";

export interface AccessGridRowModel {
  id: string;
  name: string;
  mark: ReactNode;
  // A muted line under the name for a row that needs a look: Owners only, Hidden from 2 roles.
  meta?: string | undefined;
  trailing?: ReactNode;
}

interface AccessGridRowProps {
  row: AccessGridRowModel;
  roles: AccessGridRole[];
  isOn: (rowId: string, roleId: string) => boolean;
  onToggle: (rowId: string, roleId: string) => void;
  trailing: boolean;
}

export const AccessGridRow = ({ row, roles, isOn, onToggle, trailing }: AccessGridRowProps) => (
  <tr className="group/row">
    <th scope="row" className={cn(stickyCell, stickyHover, "border-b border-border py-2 pr-3 pl-6 text-left font-normal")}>
      <span className="flex min-w-0 items-center gap-2.5">
        {row.mark}
        <span className="min-w-0">
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="block max-w-36 truncate @2xl:max-w-56">{row.name}</span>
            </TooltipTrigger>
            <TooltipContent>{row.name}</TooltipContent>
          </Tooltip>
          {row.meta && <span className="block font-mono text-[11px] whitespace-nowrap text-muted-foreground">{row.meta}</span>}
        </span>
      </span>
    </th>
    {roles.map((role) => (
      <td key={role.id} className="border-b border-border px-1.5 text-center group-hover/row:bg-accent/40">
        <label className="inline-grid size-9 cursor-pointer place-items-center rounded-md hover:bg-accent">
          <Checkbox checked={isOn(row.id, role.id)} onCheckedChange={() => onToggle(row.id, role.id)} aria-label={`${role.name} in ${row.name}`} />
        </label>
      </td>
    ))}
    {trailing && <td className="border-b border-border py-1 pr-4 pl-3 group-hover/row:bg-accent/40">{row.trailing}</td>}
  </tr>
);

import { PlusIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import type { Computer } from "@/models/Pairing";

interface AddComputerRowProps {
  // The computers that can still be added.
  computers: Computer[];
  // Listed beside them, greyed out: computers that cannot be picked.
  unavailable: { id: string; label: string }[];
  onAdd: (id: string) => void;
}

// The last row of a ranked list: a menu of the computers not in it yet.
export const AddComputerRow = ({ computers, unavailable, onAdd }: AddComputerRowProps) => (
  <li className="px-1.5 py-1">
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" variant="ghost" size="sm" className="text-muted-foreground hover:text-foreground">
          <PlusIcon aria-hidden />
          Add computer
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="max-w-xs min-w-52">
        {computers.map((c) => (
          <DropdownMenuItem key={c.id} onSelect={() => onAdd(c.id)} title={c.name}>
            <span className="truncate">{c.name}</span>
          </DropdownMenuItem>
        ))}
        {unavailable.map((c) => (
          <DropdownMenuItem key={c.id} disabled title={c.label}>
            <span className="truncate">{c.label}</span>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  </li>
);

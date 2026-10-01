import type { ReactNode } from "react";
import { ChevronDownIcon } from "lucide-react";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { usePointerCloseFocus } from "@/hooks/usePointerCloseFocus";
import { LEVEL_HINTS, levelName } from "@/models/PermissionLevel";

interface LevelDropdownProps {
  label: string;
  levelCount: number;
  level: number | undefined;
  onLevel: (level: number) => void;
  // What the trigger reads when it is not the bare level ("Write + Clone", "Custom").
  display?: string | undefined;
  // Read-only: the value without a menu.
  disabled?: boolean;
  children?: ReactNode;
}

// A muted value trailing its row; the ladder, the domain's verbs, and the row's actions all live in its one menu.
export const LevelDropdown = ({ label, levelCount, level, onLevel, display, disabled = false, children }: LevelDropdownProps) => {
  const shown = display ?? levelName(level ?? 0);
  const { triggerProps, contentProps } = usePointerCloseFocus<HTMLButtonElement>();
  return (
    <>
      {disabled && <span className="flex h-8 max-w-48 shrink-0 items-center px-2 text-sm text-muted-foreground"><span className="truncate">{shown}</span></span>}
      {!disabled && (
        <DropdownMenu>
          <DropdownMenuTrigger
            {...triggerProps}
            aria-label={`${label}: ${shown}`}
            className="flex h-8 max-w-48 shrink-0 items-center gap-1 rounded-md px-2 text-sm text-muted-foreground outline-none transition-colors duration-150 ease-standard hover:bg-accent/60 hover:text-foreground focus-visible:ring-[3px] focus-visible:ring-ring/30 data-[state=open]:bg-accent data-[state=open]:text-foreground"
          >
            <span className="truncate">{shown}</span>
            <ChevronDownIcon className="size-3.5 shrink-0 opacity-70" aria-hidden />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56" {...contentProps}>
            <DropdownMenuRadioGroup value={level === undefined ? "" : String(level)} onValueChange={(next) => onLevel(Number(next))}>
              {Array.from({ length: levelCount + 1 }, (_, rung) => (
                <DropdownMenuRadioItem key={rung} value={String(rung)}>
                  <span className="flex flex-col">
                    <span>{levelName(rung)}</span>
                    <span className="text-xs text-muted-foreground">{LEVEL_HINTS[rung]}</span>
                  </span>
                </DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
            {children}
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </>
  );
};

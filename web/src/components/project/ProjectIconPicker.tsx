import { useState } from "react";

import { ProjectIcon } from "@/components/project/ProjectIcon";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { PROJECT_ICON_NAMES } from "@/models/Project";

interface ProjectIconPickerProps {
  icon: string;
  prefix: string;
  onChange: (icon: string) => void;
}

const optionClass = (active: boolean) =>
  cn(
    "flex size-9 items-center justify-center rounded-md text-muted-foreground transition-colors duration-150 ease-standard hover:bg-accent hover:text-foreground",
    active && "bg-accent text-foreground",
  );

// "None" falls back to the prefix-letter/Box placeholder rendered by ProjectIcon.
export const ProjectIconPicker = ({ icon, prefix, onChange }: ProjectIconPickerProps) => {
  const [open, setOpen] = useState(false);

  const select = (next: string) => {
    onChange(next);
    setOpen(false);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button type="button" variant="outline" size="icon" aria-label="Change project icon">
          <ProjectIcon icon={icon} prefix={prefix} className="size-4" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto">
        <div className="grid grid-cols-4 gap-1">
          <button
            type="button"
            aria-label="None"
            aria-pressed={icon === ""}
            className={cn(optionClass(icon === ""), "text-[11px] font-medium tracking-wide")}
            onClick={() => select("")}
          >
            None
          </button>
          {PROJECT_ICON_NAMES.map((name) => (
            <button
              key={name}
              type="button"
              aria-label={name}
              aria-pressed={icon === name}
              className={optionClass(icon === name)}
              onClick={() => select(name)}
            >
              <ProjectIcon icon={name} className="size-4" />
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
};

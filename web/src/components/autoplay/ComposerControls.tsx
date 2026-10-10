import { PlusIcon, XIcon, type LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { ValueOption } from "@/hooks/AutoPlayValueHooks";
import { cn } from "@/lib/utils";

interface InlineSelectProps {
  label: string;
  value: string;
  options: ValueOption[];
  onChange: (value: string) => void;
  disabled?: boolean | undefined;
  className?: string;
}

// A select sized to its value, so a line of them reads as a sentence.
export const InlineSelect = ({ label, value, options, onChange, disabled, className }: InlineSelectProps) => (
  <Select value={value} onValueChange={onChange} disabled={disabled ?? false}>
    <SelectTrigger aria-label={label} className={cn("h-8 w-auto gap-1.5 px-2.5", className)}>
      <SelectValue />
    </SelectTrigger>
    <SelectContent>
      {options.map((option) => (
        <SelectItem key={option.value} value={option.value}>
          {option.label}
        </SelectItem>
      ))}
    </SelectContent>
  </Select>
);

interface QuietIconProps {
  label: string;
  onClick: () => void;
  icon?: LucideIcon;
}

export const QuietIcon = ({ label, onClick, icon: Icon = PlusIcon }: QuietIconProps) => (
  <Tooltip>
    <TooltipTrigger asChild>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        aria-label={label}
        onClick={onClick}
        className="size-7 shrink-0 text-muted-foreground hover:text-foreground"
      >
        <Icon className="size-3.5" />
      </Button>
    </TooltipTrigger>
    <TooltipContent>{label}</TooltipContent>
  </Tooltip>
);

export const RemoveIcon = ({ label, onClick }: { label: string; onClick: () => void }) => (
  <QuietIcon label={label} onClick={onClick} icon={XIcon} />
);

export const ComposerSection = ({ title, children }: { title: string; children: ReactNode }) => (
  <section className="space-y-2" aria-label={title}>
    <h3 className="text-sm font-medium">{title}</h3>
    {children}
  </section>
);

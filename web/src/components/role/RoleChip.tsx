import { cn } from "@/lib/utils";

// A role is a tag: a neutral hairline pill, never tinted or the accent; 1px top padding sets its text on the meta line's baseline.
export const RoleChip = ({ name, className }: { name: string; className?: string }) => (
  <span
    title={name}
    className={cn("inline-flex h-5 max-w-48 min-w-0 items-center rounded-full border border-border px-2 pt-px text-micro leading-none font-medium text-muted-foreground", className)}
  >
    <span className="truncate">{name}</span>
  </span>
);

export const NoRolesChip = () => (
  <span className="inline-flex h-5 shrink-0 items-center rounded-full border border-dashed border-foreground/25 px-2 text-micro leading-none text-muted-foreground">
    No roles
  </span>
);

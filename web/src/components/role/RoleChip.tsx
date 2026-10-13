// A role is a tag, not a state: a neutral hairline pill, never tinted, never the accent.
export const RoleChip = ({ name }: { name: string }) => (
  <span
    title={name}
    className="inline-flex h-5 max-w-40 min-w-0 items-center rounded-full border border-border bg-foreground/[0.04] px-2 text-[11px] leading-none font-medium text-foreground/85"
  >
    <span className="truncate">{name}</span>
  </span>
);

export const NoRolesChip = () => (
  <span className="inline-flex h-5 shrink-0 items-center rounded-full border border-dashed border-foreground/25 px-2 text-[11px] leading-none text-muted-foreground">
    No roles
  </span>
);

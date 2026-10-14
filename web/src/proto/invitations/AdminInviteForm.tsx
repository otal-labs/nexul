import { useState } from "react";
import { ChevronDown, Lock, TriangleAlert } from "lucide-react";

import { Microheader } from "@/components/Microheader";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { cn } from "@/lib/utils";
import { workspaceRoles } from "@/proto/invitations/data";

type Role = (typeof workspaceRoles)[number];

// The checklist row gains the role's description under its name; a role without one keeps the single-line row.
const RoleRow = ({ role, checked, onToggle }: { role: Role; checked: boolean; onToggle: () => void }) => {
  const id = `proto-role-${role.id}`;
  return (
    <li className="flex min-h-11 items-start gap-3 py-2.5">
      <Checkbox id={id} checked={checked} disabled={Boolean(role.locked)} onCheckedChange={onToggle} className="mt-0.5" />
      <label htmlFor={id} className={cn("min-w-0 flex-1", !role.locked && "cursor-pointer")}>
        <span className={cn("block text-sm font-medium", role.locked && "text-muted-foreground")}>{role.name}</span>
        {role.description && <span className="mt-0.5 block text-xs leading-[1.125rem] text-muted-foreground">{role.description}</span>}
        {role.locked && (
          <span className="mt-1 flex items-start gap-1.5 text-xs text-muted-foreground">
            <Lock aria-hidden className="mt-px size-3 shrink-0" />
            {role.locked}
          </span>
        )}
      </label>
      {!role.locked && <span className="shrink-0 pt-0.5 font-mono text-[11px] text-muted-foreground">{role.tally}</span>}
    </li>
  );
};

export const AdminInviteForm = ({ worst }: { worst: boolean }) => {
  const [picked, setPicked] = useState<string[]>(["client"]);
  const [days, setDays] = useState("7");
  const roles: Role[] = worst
    ? workspaceRoles.map((role) =>
        role.id === "senior"
          ? { ...role, name: "Senior Developer for the customer portal and billing migration", description: `${role.description} Also approves releases and keeps the release notes for every project it reaches.` }
          : role,
      )
    : workspaceRoles;
  const givable = roles.filter((role) => !role.locked);
  const locked = roles.filter((role) => role.locked);
  const toggle = (id: string) => setPicked(picked.includes(id) ? picked.filter((x) => x !== id) : [...picked, id]);

  return (
    <div className="relative grid h-dvh place-items-center bg-background p-4">
      <div className="glass-overlay absolute inset-0" />
      <div className="glass-popover relative flex max-h-[calc(100dvh-4rem)] w-full max-w-2xl flex-col gap-4 rounded-xl p-6 shadow-overlay">
        <header className="space-y-1.5 pr-6">
          <h2 className="text-base leading-snug font-semibold">Create invitation link</h2>
          <p className="text-sm text-muted-foreground">
            For someone without an account here. Pick the workspaces and the roles the link gives, then copy it and send it yourself. It works once.
          </p>
        </header>
        <div className="-mx-6 min-h-0 flex-1 space-y-5 overflow-y-auto px-6">
          <fieldset className="space-y-2">
            <legend className="text-sm font-medium">Link expires in</legend>
            <RadioGroup value={days} onValueChange={setDays} className="grid grid-cols-2 gap-2">
              <label className="flex items-center gap-2 rounded-md border p-3 text-sm">
                <RadioGroupItem value="1" />1 day
              </label>
              <label className="flex items-center gap-2 rounded-md border p-3 text-sm">
                <RadioGroupItem value="7" />7 days
              </label>
            </RadioGroup>
          </fieldset>
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Microheader>Workspace access</Microheader>
              <button type="button" className="text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline">
                Add workspace
              </button>
            </div>
            <div className="flex min-h-11 items-center gap-2 border-b py-1.5">
              <button type="button" className="flex h-8 min-w-0 flex-1 items-center justify-between rounded-md border border-input px-3 text-sm dark:bg-input/20">
                <span className="truncate">{worst ? "Northwind Traders — Customer Portal Rebuild (2026)" : "Northwind"}</span>
                <ChevronDown className="size-4 shrink-0 opacity-50" />
              </button>
              <button type="button" className="text-sm text-muted-foreground hover:text-foreground">
                Remove
              </button>
            </div>
            <section className="space-y-1">
              <div className="flex items-baseline justify-between gap-3">
                <Microheader>Roles in {worst ? "Northwind Traders — Customer Portal Rebuild (2026)" : "Northwind"}</Microheader>
                <span className="shrink-0 font-mono text-xs text-muted-foreground tabular-nums">
                  {picked.length} of {roles.length}
                </span>
              </div>
              <p className="text-xs text-muted-foreground">Everything these roles allow, combined. Projects that list a role let its people in.</p>
              <ul className="divide-y divide-border border-b border-border">
                {givable.map((role) => (
                  <RoleRow key={role.id} role={role} checked={picked.includes(role.id)} onToggle={() => toggle(role.id)} />
                ))}
              </ul>
              <div className="pt-4">
                <Microheader>Can&apos;t give</Microheader>
                <ul className="divide-y divide-border">
                  {locked.map((role) => (
                    <RoleRow key={role.id} role={role} checked={false} onToggle={() => undefined} />
                  ))}
                </ul>
              </div>
              {picked.length === 0 && (
                <p className="settle-in flex items-start gap-2 pt-3 text-xs text-muted-foreground">
                  <TriangleAlert aria-hidden className="mt-px size-3.5 shrink-0 text-warning" />
                  <span>Pick at least one role. An invitation gives a role on joining.</span>
                </p>
              )}
            </section>
          </div>
        </div>
        <footer className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <Button type="button" variant="outline">
            Cancel
          </Button>
          <Button type="button" disabled={picked.length === 0}>
            Create link
          </Button>
        </footer>
      </div>
    </div>
  );
};

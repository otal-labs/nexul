import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { PersonAvatar } from "@/proto/invitations/parts";

const MAX = 140;
const CLIENT = "Reads and writes the tickets and docs of the projects you're added to.";
const LONG =
  "Reads every ticket, doc and deploy in the projects you're added to, moves tickets between stages and runs the release plays.";

const TABS = [
  { name: "Instance", count: "0/9" },
  { name: "Workspace", count: "2/12" },
  { name: "In projects", count: "6/20" },
];

const ROWS = [
  { name: "Tickets", sub: "Read and write", on: true },
  { name: "Docs", sub: "Read, write and comment", on: true },
  { name: "Deploys", sub: "Nothing", on: false },
];

// The role dialog with one new field under Name: a single line of what the role is for, counted against its cap.
export const RoleEditorDescription = ({ worst }: { worst: boolean }) => {
  const [name, setName] = useState("Client");
  const [description, setDescription] = useState(worst ? LONG : CLIENT);
  const dirty = description !== CLIENT || name !== "Client";
  const left = MAX - description.length;

  return (
    <div className="relative grid h-dvh place-items-center bg-background p-4">
      <div className="glass-overlay absolute inset-0" />
      <div className="glass-popover relative flex h-[min(52rem,calc(100dvh-4rem))] w-full max-w-[40rem] flex-col overflow-hidden rounded-xl shadow-overlay">
        <header className="space-y-1 px-6 pt-6 pr-12 pb-4">
          <h2 className="text-base leading-snug font-semibold">Edit {name || "role"}</h2>
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <span className="flex -space-x-1.5">
              {["alice", "bob", "sam"].map((person) => (
                <PersonAvatar key={person} name={person} className="size-5 text-[9px] ring-2 ring-popover" />
              ))}
            </span>
            Saving changes it for alice, bob and sam.
          </div>
        </header>
        <div className="space-y-4 border-t px-6 pt-5 pb-4">
          <label className="grid gap-1.5">
            <span className="text-sm font-medium">Name</span>
            <Input value={name} onChange={(event) => setName(event.target.value)} />
          </label>
          <div className="grid gap-1.5">
            <div className="flex items-baseline justify-between gap-3">
              <label htmlFor="proto-role-description" className="text-sm font-medium">
                Description
              </label>
              <span className={cn("font-mono text-[11px] tabular-nums text-muted-foreground", left < 20 && "text-warning")}>
                {description.length} / {MAX}
              </span>
            </div>
            <Input
              id="proto-role-description"
              value={description}
              maxLength={MAX}
              placeholder="What someone with this role does, in one line"
              onChange={(event) => setDescription(event.target.value)}
            />
            <p className="text-xs text-muted-foreground">Shown on invitations and wherever someone picks roles.</p>
          </div>
        </div>
        <div className="flex gap-1 border-b px-6">
          {TABS.map((tab, index) => (
            <span
              key={tab.name}
              className={cn(
                "-mb-px flex items-center gap-2 rounded-t-md border border-transparent px-3 py-2 text-sm",
                index === 2 && "border-border border-b-popover bg-popover font-medium",
                index !== 2 && "text-muted-foreground",
              )}
            >
              {tab.name}
              <span className="font-mono text-[11px] text-muted-foreground">{tab.count}</span>
            </span>
          ))}
        </div>
        <ul className="min-h-0 flex-1 divide-y divide-border overflow-y-auto px-6">
          {ROWS.map((row) => (
            <li key={row.name} className="flex min-h-12 items-center justify-between gap-3 py-2.5">
              <span>
                <span className="block text-sm font-medium">{row.name}</span>
                <span className="block text-xs text-muted-foreground">{row.sub}</span>
              </span>
              <Switch defaultChecked={row.on} aria-label={row.name} />
            </li>
          ))}
        </ul>
        <footer className="flex items-center gap-2 border-t px-6 py-4">
          <span className="mr-auto flex items-center gap-2 text-xs text-muted-foreground">
            {dirty && <span className="size-1.5 rounded-full bg-foreground" aria-hidden />}
            {dirty ? "Unsaved changes" : "No changes yet"}
          </span>
          <Button type="button" variant="outline">
            Cancel
          </Button>
          <Button type="button" disabled={!dirty}>
            Save
          </Button>
        </footer>
      </div>
    </div>
  );
};

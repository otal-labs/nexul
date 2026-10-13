import { ChevronLeftIcon, LockIcon, RocketIcon, RotateCcwIcon, ScrollTextIcon, Trash2Icon } from "lucide-react";

import { LockedButton, type Variant } from "@/proto/lockedActions/Locked";

const ROLE = "Your role can't deploy";
const RULE = "Denied on stack api for bob";

// A web-drawn stand-in for the phone's pushed Stack screen and its row action sheet, at 412pt, on the phone's solid surfaces.
export const Phone = ({ variant }: { variant: Variant }) => (
  <div className="relative mx-auto flex h-[860px] w-[412px] flex-col overflow-hidden bg-background text-foreground" data-phone>
    <div className="flex h-14 items-center gap-2 px-3 text-[17px] font-medium">
      <ChevronLeftIcon className="size-6" />
      Stack
    </div>
    <div className="px-5 pt-2">
      <p className="font-mono text-xs text-muted-foreground">Atlas</p>
      <h1 className="type-display mt-1 text-[30px]">api</h1>
      <p className="mt-1 text-sm text-muted-foreground">3 services · last deploy 4h ago</p>
      <div className="mt-5 flex items-start justify-end gap-2" data-actions>
        <LockedButton touch variant={variant} kind="outline" label="Rollback" icon={RotateCcwIcon} reason={RULE} />
        <LockedButton touch variant={variant} kind="primary" label="Deploy" icon={RocketIcon} reason={ROLE} />
      </div>
    </div>
    <div className="mx-4 mt-6 rounded-lg border border-border bg-card">
      {["web", "api", "worker"].map((name) => (
        <div key={name} className="flex min-h-12 items-center gap-3 border-b border-border px-4 last:border-b-0">
          <span className="size-2 rounded-full bg-success" />
          <span className="font-mono text-sm">{name}</span>
          <span className="ml-auto text-sm text-muted-foreground">running</span>
        </div>
      ))}
    </div>
    <div className="absolute inset-x-0 bottom-0 rounded-t-xl border-t border-border bg-popover px-2 pt-2 pb-6 shadow-overlay" data-sheet>
      <div className="mx-auto mb-2 h-1 w-9 rounded-full bg-muted-foreground/40" />
      <p className="px-3 pb-2 text-[15px] font-semibold">api</p>
      <div className="flex min-h-12 items-center gap-3 rounded-md px-3">
        <ScrollTextIcon className="size-5 text-muted-foreground" />
        <span className="text-[15px]">View logs</span>
      </div>
      <div className="flex min-h-14 items-start gap-3 rounded-md px-3 py-2.5 text-muted-foreground" aria-disabled>
        <Trash2Icon className="mt-0.5 size-5" />
        <span className="grid flex-1 gap-0.5">
          <span className="text-[15px]">Delete stack</span>
          {variant !== "b" && <span className="text-[13px]">{RULE}</span>}
          {variant === "b" && <span className="text-[13px] opacity-0">{RULE}</span>}
        </span>
        <LockIcon className="mt-1 size-4" />
      </div>
    </div>
  </div>
);

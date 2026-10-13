import { FileTextIcon, KeyRoundIcon, LockIcon, RocketIcon, RotateCcwIcon, RotateCwIcon, ScrollTextIcon, Trash2Icon } from "lucide-react";
import { pageTitleClassFor } from "@/components/PageHeader";
import { cn } from "@/lib/utils";

import { Button } from "@/components/ui/button";
import { ContextMenu, ContextMenuContent, ContextMenuItem, ContextMenuSeparator, ContextMenuTrigger } from "@/components/ui/context-menu";
import { microheaderClass } from "@/components/Microheader";
import {
  LockedButton,
  LockedHandle,
  LockedInlineName,
  LockedMenuItem,
  OpenHandle,
  OpenInlineName,
  type Variant,
} from "@/proto/lockedActions/Locked";

const ROLE = "Your role can't deploy";
const RULE = "Denied on stack api for bob";
const BOARD = "Your role can't change the board";
const ENV = "Your role can't edit stacks";

const SERVICES = [
  { name: "web", state: "running", reach: "app.example.com" },
  { name: "api", state: "running", reach: "api:8080" },
  { name: "worker", state: "running", reach: "no port" },
];

const COLUMNS = ["Backlog", "To do", "In progress", "In review", "Done"];

const Card = ({ title, line, children }: { title: string; line?: React.ReactNode; children: React.ReactNode }) => (
  <section className="rounded-lg border border-border bg-card p-5 shadow-card">
    <h2 className="text-[15px] font-semibold">{title}</h2>
    {line}
    <div className="mt-4">{children}</div>
  </section>
);

const ServiceRow = ({ variant, name, reach }: { variant: Variant; name: string; reach: string }) => (
  <ContextMenu>
    <ContextMenuTrigger asChild>
      <li data-row={name} className="flex items-center gap-3 border-b border-border px-1 py-2.5 last:border-b-0 hover:bg-accent/40 data-[state=open]:bg-accent">
        <span className="size-1.5 rounded-full bg-success" aria-hidden />
        <span className="font-mono text-sm">{name}</span>
        <span className="text-sm text-muted-foreground">running</span>
        <span className="ml-auto font-mono text-xs text-muted-foreground">{reach}</span>
      </li>
    </ContextMenuTrigger>
    <ContextMenuContent className="w-64">
      <ContextMenuItem>
        <ScrollTextIcon />
        View logs
      </ContextMenuItem>
      <ContextMenuItem>
        <RotateCwIcon />
        Restart
      </ContextMenuItem>
      <ContextMenuSeparator />
      <LockedMenuItem variant={variant} label="Delete stack" icon={Trash2Icon} reason={RULE} />
    </ContextMenuContent>
  </ContextMenu>
);

const ColumnRow = ({ variant, name, locked }: { variant: Variant; name: string; locked: boolean }) => (
  <li className="flex items-center gap-2 border-b border-border py-1.5 last:border-b-0">
    {locked && <LockedHandle variant={variant} name={name} reason={BOARD} />}
    {!locked && <OpenHandle name={name} />}
    <span className="size-1.5 rounded-full bg-info" aria-hidden />
    {locked && <LockedInlineName variant={variant} name={name} reason={BOARD} />}
    {!locked && <OpenInlineName name={name} />}
  </li>
);

const ReasonLine = ({ text }: { text: string }) => (
  <p className="mt-1 flex items-center gap-1.5 text-sm text-muted-foreground">
    <LockIcon className="size-3.5" />
    {text}
  </p>
);

export const Demo = ({ variant }: { variant: Variant }) => (
  <div className="panel mx-auto w-full max-w-5xl px-6 py-8 @container" data-variant={variant}>
    <header className="flex flex-wrap items-start gap-x-6 gap-y-4 border-b border-border pb-6">
      <div className="min-w-0 flex-1">
        <p className="font-mono text-xs text-muted-foreground">Atlas › Stacks</p>
        <h1 className={cn(pageTitleClassFor("api"), "mt-2")}>api</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          <span className="font-mono">stk_8f2a</span> · 3 services · last deploy 4h ago
        </p>
      </div>
      <div className="flex items-start gap-2" data-actions>
        <LockedButton variant={variant} kind="icon" label="Environment" icon={KeyRoundIcon} reason={ENV} />
        <LockedButton variant={variant} kind="outline" label="Rollback" icon={RotateCcwIcon} reason={RULE} />
        <LockedButton variant={variant} kind="primary" label="Deploy" icon={RocketIcon} reason={ROLE} />
      </div>
    </header>

    <div className="mt-6 grid gap-4 @3xl:grid-cols-2">
      <Card title="Services" line={<p className="mt-1 text-sm text-muted-foreground">Right-click a row for its menu.</p>}>
        <ul data-services>
          {SERVICES.map((service) => (
            <ServiceRow key={service.name} variant={variant} name={service.name} reach={service.reach} />
          ))}
        </ul>
      </Card>
      <Card
        title="Board columns"
        line={variant === "c" ? <ReasonLine text={BOARD} /> : <p className="mt-1 text-sm text-muted-foreground">Drag to reorder, click a name to rename.</p>}
      >
        <ul data-columns>
          {COLUMNS.map((name, index) => (
            <ColumnRow key={name} variant={variant} name={name} locked={index > 0} />
          ))}
        </ul>
        <p className="mt-3 text-xs text-muted-foreground">First row unlocked, for comparison.</p>
      </Card>
    </div>

    <section className="mt-4 rounded-lg border border-border bg-card p-5 shadow-card" data-compare>
      <h2 className="text-[15px] font-semibold">One action, five states</h2>
      <div className="mt-4 grid grid-cols-2 gap-x-6 gap-y-5 @3xl:grid-cols-5">
        <figure className="grid content-start gap-2">
          <figcaption className={microheaderClass}>Ready</figcaption>
          <div>
            <Button>
              <RocketIcon />
              Deploy
            </Button>
          </div>
        </figure>
        <figure className="grid content-start gap-2">
          <figcaption className={microheaderClass}>Loading</figcaption>
          <div>
            <Button loading>
              <RocketIcon />
              Deploy
            </Button>
          </div>
        </figure>
        <figure className="grid content-start gap-2">
          <figcaption className={microheaderClass}>Locked, role</figcaption>
          <div>
            <LockedButton variant={variant} kind="primary" label="Deploy" icon={RocketIcon} reason={ROLE} />
          </div>
        </figure>
        <figure className="grid content-start gap-2">
          <figcaption className={microheaderClass}>Locked, rule</figcaption>
          <div>
            <LockedButton variant={variant} kind="primary" label="Deploy" icon={RocketIcon} reason={RULE} />
          </div>
        </figure>
        <figure className="grid content-start gap-2">
          <figcaption className={microheaderClass}>Locked pill</figcaption>
          <p className="text-sm">
            See{" "}
            <span className="mention-chip mention-chip--inert">
              <FileTextIcon className="size-[1.1em] shrink-0" />
              <span className="mention-chip__label">Pricing</span>
              <LockIcon className="size-[0.9em] shrink-0 text-muted-foreground" />
            </span>
          </p>
        </figure>
      </div>
    </section>
  </div>
);

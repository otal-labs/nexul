import { useReactFlow, useStore, useStoreApi, useViewport } from "@xyflow/react";
import {
  ArrowLeftIcon,
  ArrowRightIcon,
  LayoutGridIcon,
  LockIcon,
  LockOpenIcon,
  MaximizeIcon,
  MinusIcon,
  MoreHorizontalIcon,
  PlusIcon,
  type LucideIcon,
} from "lucide-react";

import { ActionButton } from "@/components/action/ActionButton";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { TopologySummary } from "@/components/topology/wiring";
import { useActionState } from "@/hooks/useActionState";
import { cn } from "@/lib/utils";
import { ServiceStatus, type AttentionStatus } from "@/models/Topology";
import { laneBefore, laneBeyond, panBackTo, panTo, type Lane } from "@/utils/TopologyLayout";

interface TopologyToolbarProps {
  summary: TopologySummary;
  lanes: Lane[];
  onAddExternal: () => void;
  onArrange: () => void;
  onFit: () => void;
  onFrame: (status: AttentionStatus) => void;
}

interface Tool {
  label: string;
  // The bar width from which the tool sits in the bar rather than the menu.
  shownFrom?: "@2xl" | "@5xl";
  icon: LucideIcon;
  onClick: () => void;
  // A word beside the icon on a wide bar, for a tool whose icon alone does not say what it does.
  word?: string;
  // Back points the way it goes, so its arrow leads the word.
  iconFirst?: boolean;
  disabled?: boolean;
}

const ToolButton = ({ label, icon: Icon, onClick, word, iconFirst, disabled, className }: Tool & { className?: string }) => (
  <Tooltip>
    <TooltipTrigger asChild>
      <Button variant="ghost" size={word ? "sm" : "icon"} className={cn(word ? "h-8 px-2" : "size-8", className)} aria-label={label} onClick={onClick} disabled={disabled}>
        {iconFirst && <Icon />}
        {word && <span>{word}</span>}
        {!iconFirst && <Icon />}
      </Button>
    </TooltipTrigger>
    <TooltipContent>{label}</TooltipContent>
  </Tooltip>
);

// A count that needs attention steps through those services; plain inline text keeps every count on one baseline.
const AttentionCount = ({ n, label, dot, onClick }: { n: number; label: string; dot: string; onClick: () => void }) => (
  <button
    type="button"
    className="cursor-pointer rounded-sm whitespace-nowrap underline-offset-4 hover:text-foreground hover:underline"
    aria-label={`Show the next ${label} service, of ${n}`}
    onClick={onClick}
  >
    <span className={cn("relative -top-px mr-1.5 inline-block size-1.5 rounded-full align-middle", dot)} aria-hidden />
    <span className="tabular-nums text-foreground">{n}</span> {label}
  </button>
);

// Plain inline text, so every count sits on the same baseline whether or not a dot leads it.
const Count = ({ n, label, dot, className }: { n: number; label: string; dot?: string; className?: string }) => (
  <span className={cn("whitespace-nowrap", className)}>
    {dot && <span className={cn("relative -top-px mr-1.5 inline-block size-1.5 rounded-full align-middle", dot)} aria-hidden />}
    <span className="tabular-nums text-foreground">{n}</span> {label}
  </span>
);

// Where the bar is narrower than 64rem the rarer tools fold into one menu (More from 42rem), so it keeps to one line.
const MoreTools = ({ tools }: { tools: Tool[] }) => (
  <DropdownMenu>
    <DropdownMenuTrigger asChild>
      <Button variant="ghost" size="icon" className="size-8 @5xl:hidden" aria-label="More tools">
        <MoreHorizontalIcon />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end">
      {tools.map((t) => (
        <DropdownMenuItem key={t.label} onSelect={t.onClick} disabled={t.disabled}>
          <t.icon />
          {t.label}
        </DropdownMenuItem>
      ))}
    </DropdownMenuContent>
  </DropdownMenu>
);

// The canvas's one bar, which is also the page's header.
export const TopologyToolbar = ({ summary, lanes, onAddExternal, onArrange, onFit, onFrame }: TopologyToolbarProps) => {
  const { zoomIn, zoomOut, zoomTo, setViewport } = useReactFlow();
  const viewport = useViewport();
  const width = useStore((s) => s.width);
  const store = useStoreApi();
  const locked = useStore((s) => !s.nodesDraggable && !s.elementsSelectable);
  const canWrite = useActionState({ permission: "topology:write" }).state === "live";
  const beyond = laneBeyond(lanes, viewport, width);
  const before = laneBefore(lanes, viewport);
  const rare: Tool[] = [
    {
      label: "Show earlier lanes",
      word: "Back",
      iconFirst: true,
      shownFrom: "@2xl" as const,
      icon: ArrowLeftIcon,
      disabled: !before,
      onClick: () => before && void setViewport(panBackTo(before, viewport)),
    },
    // Always in the bar, so the tools beside it never shift; at the last lane it rests disabled.
    {
      label: "Show more lanes",
      word: "More",
      shownFrom: "@2xl" as const,
      icon: ArrowRightIcon,
      disabled: !beyond,
      onClick: () => beyond && void setViewport(panTo(beyond, viewport, width)),
    },
    {
      label: locked ? "Unlock the map" : "Lock the map",
      icon: locked ? LockIcon : LockOpenIcon,
      onClick: () => store.setState({ nodesDraggable: locked, nodesConnectable: locked, elementsSelectable: locked }),
    },
    ...(canWrite ? [{ label: "Arrange in lanes", word: "Arrange", icon: LayoutGridIcon, onClick: onArrange }] : []),
  ];
  return (
    <div className="@container border-b border-border">
      <div className="flex min-h-12 items-center gap-x-4 px-3 py-2">
      <div className="flex min-w-0 items-baseline gap-x-4">
        <h1 className="type-display shrink-0 text-xl">Topology</h1>
        <div className="flex min-w-0 items-baseline gap-x-3 text-xs text-muted-foreground">
          <Count n={summary.services} label="services" className="hidden @5xl:inline" />
          {summary.failed > 0 && <AttentionCount n={summary.failed} label="failed" dot="bg-destructive" onClick={() => onFrame(ServiceStatus.Failed)} />}
          {summary.stopped > 0 && <AttentionCount n={summary.stopped} label="stopped" dot="bg-warning" onClick={() => onFrame(ServiceStatus.Stopped)} />}
          <Count n={summary.hostnames} label="hostnames" className="hidden @5xl:inline" />
        </div>
      </div>
      <div className="ml-auto flex shrink-0 items-center gap-1">
        <ToolButton label="Zoom out" icon={MinusIcon} onClick={() => void zoomOut()} />
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" className="h-8 w-14 px-0 font-mono text-xs tabular-nums" aria-label="Zoom to 100%" onClick={() => void zoomTo(1)}>
              {Math.round(viewport.zoom * 100)}%
            </Button>
          </TooltipTrigger>
          <TooltipContent>Zoom to 100%</TooltipContent>
        </Tooltip>
        <ToolButton label="Zoom in" icon={PlusIcon} onClick={() => void zoomIn()} />
        <ToolButton label="Fit" icon={MaximizeIcon} onClick={onFit} />
        {rare.map((t) => (
          <ToolButton key={t.label} {...t} className={t.shownFrom === "@2xl" ? "hidden @2xl:inline-flex" : "hidden @5xl:inline-flex"} />
        ))}
        <MoreTools tools={rare} />
        <ActionButton
          permission="topology:write"
          verb="add topology nodes"
          variant="outline"
          size="sm"
          icon={PlusIcon}
          className="ml-2"
          aria-label="Add external node"
          onClick={onAddExternal}
        >
          External
        </ActionButton>
      </div>
      </div>
    </div>
  );
};

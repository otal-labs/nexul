import { Boxes, FileCode, FileText, Folder, NotebookPen, TextQuote } from "lucide-react";

import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import type { ProtoSource, SourceKind, Stance } from "@/components/memory/prototype/SourcesProtoData";
import { cn } from "@/lib/utils";

export const KIND_LABEL: Record<SourceKind, string> = { path: "Path", doc: "Doc", memory: "Memory", project: "Project", text: "Paste text" };

const KIND_ICON = { doc: FileText, memory: NotebookPen, project: Boxes, text: TextQuote } as const;

export const SourceKindIcon = ({ source, className }: { source: Pick<ProtoSource, "kind" | "name">; className?: string }) => {
  const folder = source.kind === "path" && source.name.endsWith("/");
  const Icon = source.kind === "path" ? (folder ? Folder : FileCode) : KIND_ICON[source.kind];
  return <Icon className={cn("size-4 shrink-0 text-muted-foreground", className)} aria-label={KIND_LABEL[source.kind]} role="img" />;
};

interface SourceStanceProps {
  label: string;
  value: Stance;
  onChange: (value: Stance) => void;
  disabled?: boolean;
  size?: "xs" | "sm";
}

const itemClass = {
  xs: "h-6 px-2 text-[11px]",
  sm: "h-7 px-3 text-xs",
};

// Follow | Question as a segmented pair, the Every project row's control at row size.
export const SourceStance = ({ label, value, onChange, disabled = false, size = "xs" }: SourceStanceProps) => (
  <ToggleGroup
    type="single"
    variant="outline"
    size="sm"
    role="radiogroup"
    aria-label={`Stance for ${label}`}
    disabled={disabled}
    value={value}
    onValueChange={(next) => next && onChange(next as Stance)}
    className="shrink-0"
  >
    {(["follow", "question"] as const).map((stance) => (
      <ToggleGroupItem
        key={stance}
        value={stance}
        className={cn(
          itemClass[size],
          "min-w-0 text-muted-foreground transition-colors duration-150 ease-standard data-[state=on]:bg-accent data-[state=on]:text-foreground",
        )}
      >
        {stance === "follow" ? "Follow" : "Question"}
      </ToggleGroupItem>
    ))}
  </ToggleGroup>
);

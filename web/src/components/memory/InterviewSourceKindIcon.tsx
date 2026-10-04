import { Boxes, FileCode, FileText, Folder, NotebookPen, TextQuote } from "lucide-react";

import { SOURCE_KIND_LABEL, type SourceKind } from "@/models/InterviewSource";

interface InterviewSourceKindIconProps {
  kind: SourceKind;
  // A path ending in "/", or whose last part has no extension, shows as a folder; the server stores paths without the slash.
  name: string;
}

const KIND_ICON = { doc: FileText, memory: NotebookPen, project: Boxes, text: TextQuote } as const;

export const InterviewSourceKindIcon = ({ kind, name }: InterviewSourceKindIconProps) => {
  const folder = name.endsWith("/") || !(name.split("/").at(-1) ?? "").includes(".");
  const pathIcon = folder ? Folder : FileCode;
  const Icon = kind === "path" ? pathIcon : KIND_ICON[kind];
  return <Icon className="size-4 shrink-0 text-muted-foreground" role="img" aria-label={SOURCE_KIND_LABEL[kind]} />;
};

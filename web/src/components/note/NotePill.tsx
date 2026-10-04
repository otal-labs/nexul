import { FileText } from "lucide-react";

import { DialogPill } from "@/components/DialogPill";
import { formatBytes, type Attachment } from "@/models/Attachment";

interface NotePillProps {
  file: Attachment;
  onOpen: () => void;
}

export const NotePill = ({ file, onOpen }: NotePillProps) => (
  <DialogPill
    icon={<FileText className="size-3 text-muted-foreground" aria-hidden />}
    label={file.name}
    trailing={<span className="shrink-0 font-mono text-[11px] text-muted-foreground tabular-nums">{formatBytes(file.size)}</span>}
    onOpen={onOpen}
  />
);

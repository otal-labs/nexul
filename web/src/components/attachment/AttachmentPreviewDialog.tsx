import { DownloadIcon } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";

import { AttachmentCsvPreview } from "@/components/attachment/AttachmentCsvPreview";
import { AttachmentMediaPreview } from "@/components/attachment/AttachmentMediaPreview";
import { EmptyRow } from "@/components/EmptyRow";
import { downloadAttachment } from "@/hooks/AttachmentHooks";
import { formatBytes, previewKind, type Attachment } from "@/models/Attachment";

interface AttachmentPreviewDialogProps {
  attachment: Attachment;
  // The trigger; the dialog's content, and so its download, mounts only once it opens.
  children: ReactNode;
}

export const AttachmentPreviewDialog = ({ attachment, children }: AttachmentPreviewDialogProps) => {
  const kind = previewKind(attachment);
  return (
    <Dialog>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent aria-describedby={undefined} className="gap-3 p-4 sm:max-w-[min(64rem,calc(100%-2rem))]">
        <DialogHeader className="flex-row items-center gap-3 pr-8">
          <DialogTitle className="min-w-0 flex-1 truncate text-sm font-medium">{attachment.name}</DialogTitle>
          <span className="shrink-0 font-mono text-xs text-muted-foreground tabular-nums">{formatBytes(attachment.size)}</span>
          <Button type="button" variant="outline" size="sm" onClick={() => void downloadAttachment(attachment)}>
            <DownloadIcon aria-hidden />
            Download
          </Button>
        </DialogHeader>
        {(kind === "image" || kind === "svg" || kind === "pdf" || kind === "audio") && (
          <AttachmentMediaPreview attachment={attachment} kind={kind} />
        )}
        {kind === "csv" && <AttachmentCsvPreview attachment={attachment} />}
        {kind === "too-large" && <EmptyRow flush>Too large to preview here. Download it to open it.</EmptyRow>}
        {kind === "none" && <EmptyRow flush>No preview for this type of file. Download it to open it.</EmptyRow>}
      </DialogContent>
    </Dialog>
  );
};

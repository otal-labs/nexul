import { DownloadIcon, FileIcon, Trash2Icon } from "lucide-react";

import { downloadAttachment, useAttachmentBlob, useDeleteAttachment } from "@/hooks/AttachmentHooks";
import { useConfirmationDialog } from "@/hooks/useConfirmationDialog";
import { attachmentPath, formatBytes, isInlineImage, type Attachment } from "@/models/Attachment";

interface AttachmentRowProps {
  attachment: Attachment;
}

// Collapsed until the pill is hovered or focused, so a resting pill is only thumb, name, and size.
const actionsClass =
  "flex max-w-0 items-center gap-0.5 overflow-hidden opacity-0 transition-all duration-150 ease-standard group-hover/pill:max-w-12 group-hover/pill:opacity-100 group-focus-within/pill:max-w-12 group-focus-within/pill:opacity-100";
const iconButtonClass =
  "flex size-5 shrink-0 items-center justify-center rounded-full text-muted-foreground transition-colors duration-150 ease-standard hover:bg-muted/60 hover:text-foreground";

export const AttachmentRow = ({ attachment }: AttachmentRowProps) => {
  const inline = isInlineImage(attachment.content_type);
  const { data: thumb } = useAttachmentBlob(inline ? attachmentPath(attachment.id) : null);
  const deleteAttachment = useDeleteAttachment();
  const { open: confirmDelete } = useConfirmationDialog();

  const onDelete = async () => {
    const ok = await confirmDelete({ message: `Delete ${attachment.name}? Bodies that show it will lose the image.` });
    if (ok) deleteAttachment.mutate(attachment.id);
  };

  return (
    <li className="group/pill inline-flex max-w-full items-center gap-1.5 rounded-full border border-border bg-card py-0.5 pr-1 pl-1 text-xs">
      <span className="flex size-5 shrink-0 items-center justify-center overflow-hidden rounded-full bg-muted/60">
        {thumb && <img src={thumb} alt="" className="size-full object-cover" />}
        {!thumb && <FileIcon className="size-3 text-muted-foreground" aria-hidden />}
      </span>
      <span className="truncate text-foreground">{attachment.name}</span>
      <span className="shrink-0 font-mono text-[11px] text-muted-foreground tabular-nums">
        {formatBytes(attachment.size)}
      </span>
      <span className={actionsClass}>
        <button
          type="button"
          aria-label={`Download ${attachment.name}`}
          className={iconButtonClass}
          onClick={() => void downloadAttachment(attachment)}
        >
          <DownloadIcon className="size-3" aria-hidden />
        </button>
        <button
          type="button"
          aria-label={`Delete ${attachment.name}`}
          className={iconButtonClass}
          disabled={deleteAttachment.isPending}
          onClick={() => void onDelete()}
        >
          <Trash2Icon className="size-3" aria-hidden />
        </button>
      </span>
    </li>
  );
};

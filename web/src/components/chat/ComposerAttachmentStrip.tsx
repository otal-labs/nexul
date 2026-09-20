import { ComposerAttachmentThumb } from "@/components/chat/ComposerAttachmentThumb";
import type { PendingAttachment } from "@/hooks/ComposerAttachmentHooks";

interface ComposerAttachmentStripProps {
  pending: PendingAttachment[];
  onRemove: (attachmentId: string) => void;
}

export const ComposerAttachmentStrip = ({ pending, onRemove }: ComposerAttachmentStripProps) => (
  <div className="flex gap-2 overflow-x-auto px-1 pb-2">
    {pending.map(({ attachment, previewUrl }) => (
      <ComposerAttachmentThumb
        key={attachment.id}
        attachment={attachment}
        previewUrl={previewUrl}
        onRemove={() => onRemove(attachment.id)}
      />
    ))}
  </div>
);

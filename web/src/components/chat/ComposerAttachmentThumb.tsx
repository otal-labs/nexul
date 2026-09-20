import { XIcon } from "lucide-react";

import type { Attachment } from "@/models/Attachment";

interface ComposerAttachmentThumbProps {
  attachment: Attachment;
  previewUrl: string;
  onRemove: () => void;
}

export const ComposerAttachmentThumb = ({ attachment, previewUrl, onRemove }: ComposerAttachmentThumbProps) => (
  <div className="relative size-14 shrink-0 overflow-hidden rounded-md border border-border">
    <img src={previewUrl} alt={attachment.name} className="size-full object-cover" />
    <button
      type="button"
      aria-label={`Remove ${attachment.name}`}
      className="absolute top-0.5 right-0.5 flex size-4 items-center justify-center rounded-full bg-background/80 text-foreground"
      onClick={onRemove}
    >
      <XIcon className="size-3" aria-hidden />
    </button>
  </div>
);

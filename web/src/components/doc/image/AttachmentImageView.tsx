import { NodeViewWrapper, type NodeViewProps } from "@tiptap/react";

import { useAttachmentBlob } from "@/hooks/AttachmentHooks";
import { isAttachmentPath } from "@/models/Attachment";
import { cn } from "@/lib/utils";

const attachmentSrcOf = (rawSrc: string) => (isAttachmentPath(rawSrc) ? rawSrc : null);

const resolvedSrcOf = (attachmentSrc: string | null, blobUrl: string | undefined, rawSrc: string) =>
  attachmentSrc ? blobUrl : rawSrc;

// Attachment sources load through the authenticated client (useAttachmentBlob); other URLs render as-is.
export const AttachmentImageView = ({ node, selected }: NodeViewProps) => {
  const rawSrc = (node.attrs.src as string | null) ?? "";
  const attachmentSrc = attachmentSrcOf(rawSrc);
  const { data: blobUrl, error } = useAttachmentBlob(attachmentSrc);
  const resolved = resolvedSrcOf(attachmentSrc, blobUrl, rawSrc);
  const alt = (node.attrs.alt as string | null) ?? "";
  return (
    <NodeViewWrapper as="figure" className={cn("doc-image", selected && "is-selected")} data-drag-handle>
      {error && (
        <span role="img" aria-label={alt || "Image unavailable"} className="doc-image-missing">
          Image unavailable
        </span>
      )}
      {!error && resolved && <img src={resolved} alt={alt} title={(node.attrs.title as string | null) ?? undefined} />}
      {!error && !resolved && <span className="doc-image-loading" role="status" aria-label="Loading image" />}
    </NodeViewWrapper>
  );
};

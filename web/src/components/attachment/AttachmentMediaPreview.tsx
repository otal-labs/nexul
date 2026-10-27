import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useAttachmentBlob } from "@/hooks/AttachmentHooks";
import { attachmentPath, type Attachment } from "@/models/Attachment";

interface AttachmentMediaPreviewProps {
  attachment: Attachment;
  kind: "image" | "svg" | "pdf" | "audio";
}

// An SVG only ever loads through <img>, which never runs its scripts; a PDF is labelled as one so the frame can only show the viewer.
const blobTypes: Partial<Record<AttachmentMediaPreviewProps["kind"], string>> = { svg: "image/svg+xml", pdf: "application/pdf" };

export const AttachmentMediaPreview = ({ attachment, kind }: AttachmentMediaPreviewProps) => {
  const { data: url, error, isPending } = useAttachmentBlob(attachmentPath(attachment.id), blobTypes[kind]);

  return (
    <div className="min-h-0">
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {url && (kind === "image" || kind === "svg") && (
        <img src={url} alt={attachment.name} className="mx-auto max-h-[75vh] max-w-full rounded-md object-contain" />
      )}
      {url && kind === "pdf" && <iframe src={url} title={attachment.name} className="h-[75vh] w-full rounded-md border border-border bg-surface-2" />}
      {url && kind === "audio" && <audio src={url} controls aria-label={`Play ${attachment.name}`} className="w-full" />}
    </div>
  );
};

import { useAttachmentBlob } from "@/hooks/AttachmentHooks";

interface MessageImageProps {
  src: string;
  alt: string;
}

// Bytes load through the authenticated client (see useAttachmentBlob); a bare <img src> would 401.
export const MessageImage = ({ src, alt }: MessageImageProps) => {
  const { data: blobUrl, error } = useAttachmentBlob(src);
  return (
    <span className="block">
      {error && (
        <span
          role="img"
          aria-label={alt || "Image unavailable"}
          className="flex h-24 max-w-full items-center justify-center rounded-md border border-dashed border-border px-4 text-xs text-muted-foreground"
        >
          Image unavailable
        </span>
      )}
      {!error && blobUrl && (
        <a href={blobUrl} target="_blank" rel="noreferrer" className="inline-block">
          <img src={blobUrl} alt={alt} className="max-h-64 max-w-full rounded-md border border-border" />
        </a>
      )}
      {!error && !blobUrl && (
        <span
          role="status"
          aria-label="Loading image"
          className="flex h-24 w-40 max-w-full items-center justify-center rounded-md border border-dashed border-border text-xs text-muted-foreground"
        />
      )}
    </span>
  );
};

import { PlusIcon } from "lucide-react";
import { useRef } from "react";

import { AttachmentRow } from "@/components/attachment/AttachmentRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { useFetchAttachments, useUploadAttachment } from "@/hooks/AttachmentHooks";
import { cn } from "@/lib/utils";
import type { AttachmentOwner } from "@/models/Attachment";

interface AttachmentsSectionProps {
  owner: AttachmentOwner;
  className?: string;
}

// Everything pasted into the body shows up here too; files render as hover-revealing pills.
export const AttachmentsSection = ({ owner, className }: AttachmentsSectionProps) => {
  const { data, error, isPending } = useFetchAttachments(owner);
  const upload = useUploadAttachment();
  const inputRef = useRef<HTMLInputElement | null>(null);

  return (
    <section className={cn("space-y-2", className)} aria-label="Attachments">
      <div className="flex items-center justify-between">
        <h2 className="font-mono text-[11px] font-semibold tracking-[0.08em] text-muted-foreground/80 uppercase">
          Attachments
          {data && data.length > 0 && <span className="ml-1.5 tabular-nums">{data.length}</span>}
        </h2>
        <button
          type="button"
          aria-label="Add attachment"
          className="flex size-5 items-center justify-center rounded-md text-muted-foreground transition-colors duration-150 ease-standard hover:bg-muted/50 hover:text-foreground"
          disabled={upload.isPending}
          onClick={() => inputRef.current?.click()}
        >
          <PlusIcon className="size-3.5" aria-hidden />
        </button>
        <input
          ref={inputRef}
          type="file"
          multiple
          className="sr-only"
          aria-label="Attachment file"
          onChange={(e) => {
            for (const file of Array.from(e.target.files ?? [])) upload.mutate({ owner, file });
            e.target.value = "";
          }}
        />
      </div>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && data.length === 0 && <p className="text-xs text-muted-foreground">No files yet.</p>}
      {data && data.length > 0 && (
        <ul className="flex flex-wrap gap-1.5">
          {data.map((attachment) => (
            <AttachmentRow key={attachment.id} attachment={attachment} />
          ))}
        </ul>
      )}
    </section>
  );
};

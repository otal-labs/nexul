import { PlusIcon } from "lucide-react";
import { useRef } from "react";

import { AttachmentRow } from "@/components/attachment/AttachmentRow";
import { EmptyRow } from "@/components/EmptyRow";
import { ErrorDisplay } from "@/components/ErrorDisplay";
import { microheaderClass } from "@/components/Microheader";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import { microheaderClass } from "@/components/Microheader";
import { useFetchAttachments, useUploadAttachment } from "@/hooks/AttachmentHooks";
import { useHasPermission } from "@/hooks/WorkspaceHooks";
import { cn } from "@/lib/utils";
import type { AttachmentOwner } from "@/models/Attachment";

interface AttachmentsSectionProps {
  owner: SectionOwner;
  className?: string;
  actionPlacement?: "beside-label" | "end";
}

// Chat attachments take only read access to the conversation, so no section shows for one.
type SectionOwner = Exclude<AttachmentOwner, { conversation_id: string }>;

const writePermission = (owner: SectionOwner): string => {
  if ("doc_id" in owner) return "docs:write";
  if ("ticket_id" in owner) return "tickets:write";
  return "memories:write";
};

// Everything pasted into the body shows up here too; files render as hover-revealing pills.
export const AttachmentsSection = ({ owner, className, actionPlacement = "beside-label" }: AttachmentsSectionProps) => {
  const { data, error, isPending } = useFetchAttachments(owner);
  const upload = useUploadAttachment();
  const canWrite = useHasPermission(writePermission(owner));
  const inputRef = useRef<HTMLInputElement | null>(null);

  return (
    <section className={cn("space-y-2", className)} aria-label="Attachments">
      <div className={cn("flex items-center gap-1.5", actionPlacement === "end" && "justify-between")}>
        <h2 className={microheaderClass}>
          Attachments
          {data && data.length > 0 && <span className="ml-1.5 tabular-nums">{data.length}</span>}
        </h2>
        {canWrite && (
          <>
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
          </>
        )}
      </div>
      {isPending && <LoadingDisplay />}
      {error && <ErrorDisplay error={error} />}
      {data && data.length === 0 && <EmptyRow className="p-0">No files yet.</EmptyRow>}
      {data && data.length > 0 && (
        <ul className="flex flex-wrap gap-1.5">
          {data.map((attachment) => (
            <AttachmentRow key={attachment.id} attachment={attachment} canDelete={canWrite} />
          ))}
        </ul>
      )}
    </section>
  );
};

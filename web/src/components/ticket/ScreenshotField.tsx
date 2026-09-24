import { ImagePlus, X } from "lucide-react";

import { useFormDialogContext } from "@/components/dialogs/FormDialogContext";
import { useUploadAttachment } from "@/hooks/AttachmentHooks";
import type { TestFailFormData } from "@/models/TicketTest";

interface ScreenshotFieldProps {
  ticketId: string;
}

// Screenshots upload as the ticket's own attachments, so they also list under the ticket's Attachments.
export const ScreenshotField = ({ ticketId }: ScreenshotFieldProps) => {
  const { watch, setValue, getValues } = useFormDialogContext<TestFailFormData>();
  const upload = useUploadAttachment();
  const screenshots = watch("screenshots");

  const add = async (files: File[]) => {
    for (const file of files) {
      const attachment = await upload.mutateAsync({ owner: { ticket_id: ticketId }, file });
      setValue("screenshots", [...getValues("screenshots"), { id: attachment.id, name: attachment.name }]);
    }
  };

  return (
    <div className="space-y-2">
      <label className="flex h-9 w-full cursor-pointer items-center justify-center gap-2 rounded-md border border-dashed border-border text-sm text-muted-foreground transition-colors duration-150 ease-standard hover:border-ring/40 hover:text-foreground has-[:focus-visible]:ring-[3px] has-[:focus-visible]:ring-ring/30">
        <ImagePlus className="size-4" aria-hidden />
        {upload.isPending ? "Uploading…" : "Add screenshot"}
        <input
          type="file"
          accept="image/*"
          multiple
          className="sr-only"
          disabled={upload.isPending}
          onChange={(e) => {
            const files = Array.from(e.target.files ?? []);
            e.target.value = "";
            void add(files).catch(() => undefined);
          }}
        />
      </label>
      {screenshots.length > 0 && (
        <ul className="divide-y divide-border rounded-md border border-border">
          {screenshots.map((shot) => (
            <li key={shot.id} className="flex items-center gap-2 px-3 py-1.5 text-sm">
              <span className="min-w-0 flex-1 truncate font-mono text-xs">{shot.name}</span>
              <button
                type="button"
                aria-label={`Leave out ${shot.name}`}
                onClick={() => setValue("screenshots", screenshots.filter((s) => s.id !== shot.id))}
                className="flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors duration-150 ease-standard hover:bg-muted/50 hover:text-foreground"
              >
                <X className="size-3.5" aria-hidden />
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};

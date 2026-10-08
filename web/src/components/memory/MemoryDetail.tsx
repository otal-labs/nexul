import { useMemo, useState } from "react";

import { AttachmentsSection } from "@/components/attachment/AttachmentsSection";
import { DocBodyView } from "@/components/doc/DocBodyView";
import { DocTitleField } from "@/components/doc/DocTitleField";
import { formatUpdatedAgo } from "@/components/doc/docTime";
import { RichTextEditor } from "@/components/doc/RichTextEditor";
import { CloneMemoryDialog } from "@/components/memory/CloneMemoryDialog";
import { InterviewLengthMeter } from "@/components/memory/InterviewLengthMeter";
import { MemoryRail } from "@/components/memory/MemoryRail";
import { MemoryVersionsFeed } from "@/components/memory/MemoryVersionsFeed";
import { PageHeader } from "@/components/PageHeader";
import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { isDecisionsLogMemory, isInterviewMemory, type Memory } from "@/models/Memory";
import { bodyToMarkdown } from "@/utils/RichtextUtility";

interface MemoryDetailProps {
  memory: Memory;
  canWrite: boolean;
  canDelete: boolean;
  canClone: boolean;
  onSave: (input: { title: string; when_to_use: string; body: string }) => void;
  onDelete: () => void;
  saving: boolean;
}

// No live collaboration, unlike DocDetail: a memory is edited by one person at a time, saved explicitly.
export const MemoryDetail = ({ memory, canWrite, canDelete, canClone, onSave, onDelete, saving }: MemoryDetailProps) => {
  const [title, setTitle] = useState(memory.title);
  const [whenToUse, setWhenToUse] = useState(memory.when_to_use);
  const [body, setBody] = useState(memory.body);
  const [cloneOpen, setCloneOpen] = useState(false);
  const wsPath = useWorkspacePath();
  const interview = isInterviewMemory(memory);
  const decisionsLog = isDecisionsLogMemory(memory);
  const interviewLength = useMemo(() => (interview ? bodyToMarkdown(body).length : 0), [interview, body]);

  const dirty = title !== memory.title || whenToUse !== memory.when_to_use || body !== memory.body;

  return (
    <div className="@container animate-in fade-in-0 slide-in-from-bottom-1 mx-auto w-full max-w-5xl duration-200 ease-out">
      <PageHeader
        crumbs={[{ label: "Memories", to: wsPath("/memories") }]}
        title={<DocTitleField editable={canWrite} title={title} staticTitle={memory.title} onChange={setTitle} />}
        meta={<span className="font-mono text-xs tabular-nums">updated {formatUpdatedAgo(memory.updated_at)}</span>}
        actions={
          <>
            {canClone && (
              <Button variant="outline" size="sm" onClick={() => setCloneOpen(true)}>
                Clone to…
              </Button>
            )}
            {canDelete && (
              <Button variant="ghost" size="sm" onClick={onDelete}>
                Delete
              </Button>
            )}
          </>
        }
      />
      {canClone && <CloneMemoryDialog memoryId={memory.id} open={cloneOpen} onClose={() => setCloneOpen(false)} />}
      <div className="mt-6 @4xl:flex @4xl:gap-8">
        <MemoryRail memoryId={memory.id} currentVersion={memory.version} canRevert={canWrite} />

        <div className="min-w-0 flex-1">
          <article className="rounded-lg border border-border bg-card p-6 shadow-card sm:p-8">
            <div className="mx-auto max-w-3xl">
              {canWrite && (
                <div className="mb-6 space-y-3">
                  <Input
                    value={whenToUse}
                    onChange={(e) => setWhenToUse(e.target.value)}
                    placeholder="When should Agent use this? e.g. writing React code"
                    aria-label="When to use"
                    className="text-sm"
                  />
                  {interview && (
                    <p className="text-sm text-muted-foreground">
                      Always included in every agent turn in this project; it can't be switched off.
                    </p>
                  )}
                  {decisionsLog && (
                    <p className="text-sm text-muted-foreground">
                      Pulled from the memory index when an agent needs the why; never sent in every turn.
                    </p>
                  )}
                </div>
              )}
              {!canWrite && memory.when_to_use !== "" && (
                <p className="mb-6 text-sm text-muted-foreground">{memory.when_to_use}</p>
              )}

              {canWrite && <RichTextEditor value={body} onChange={setBody} aria-label="Body" attachTo={{ memory_id: memory.id }} />}
              {!canWrite && <DocBodyView body={memory.body} />}

              {canWrite && (
                <div className="mt-6 flex flex-wrap items-center justify-end gap-3">
                  {interview && <InterviewLengthMeter length={interviewLength} />}
                  <Button
                    loading={saving} disabled={!dirty}
                    onClick={() => onSave({ title, when_to_use: whenToUse, body })}
                  >
                    Save
                  </Button>
                </div>
              )}
            </div>
          </article>

          {/* The rail holds these from @4xl; both read the same queries, so there is one request. */}
          <div className="@4xl:hidden">
            <PageTabs
              label="Memory sections"
              className="mt-8"
              tabs={[
                { value: "attachments", label: "Attachments" },
                { value: "versions", label: "Versions" },
              ]}
            >
              <PageTabsContent value="attachments">
                <AttachmentsSection owner={{ memory_id: memory.id }} />
              </PageTabsContent>
              <PageTabsContent value="versions">
                <MemoryVersionsFeed memoryId={memory.id} currentVersion={memory.version} canRevert={canWrite} />
              </PageTabsContent>
            </PageTabs>
          </div>
        </div>
      </div>
    </div>
  );
};

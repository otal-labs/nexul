import { useMemo, useState } from "react";

import { DocBodyView } from "@/components/doc/DocBodyView";
import { DocTitleField } from "@/components/doc/DocTitleField";
import { RichTextEditor } from "@/components/doc/RichTextEditor";
import { CloneMemoryDialog } from "@/components/memory/CloneMemoryDialog";
import { InterviewLengthMeter } from "@/components/memory/InterviewLengthMeter";
import { MemoryActionsMenu } from "@/components/memory/MemoryActionsMenu";
import { memoryFolderOptions } from "@/components/memory/memoryGroups";
import { MemorySections } from "@/components/memory/MemorySections";
import { MemorySettingsLine } from "@/components/memory/MemorySettingsLine";
import { MemoryWhenToUse } from "@/components/memory/MemoryWhenToUse";
import { PageHeader } from "@/components/PageHeader";
import { Button } from "@/components/ui/button";
import { useActionState } from "@/hooks/useActionState";
import { useEmbeddedCrumbs } from "@/hooks/useEmbeddedCrumbs";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { isInterviewMemory, type Memory } from "@/models/Memory";
import { bodyToMarkdown } from "@/utils/RichtextUtility";

interface MemoryDetailProps {
  memory: Memory;
  onSave: (input: { title: string; when_to_use: string; body: string }) => void;
  onDelete: () => void;
  saving: boolean;
}

// No live collaboration, unlike DocDetail: a memory is edited by one person at a time, saved explicitly.
export const MemoryDetail = ({ memory, onSave, onDelete, saving }: MemoryDetailProps) => {
  const [title, setTitle] = useState(memory.title);
  const [whenToUse, setWhenToUse] = useState(memory.when_to_use);
  const [body, setBody] = useState(memory.body);
  const [cloneOpen, setCloneOpen] = useState(false);
  const canWrite = useActionState({ permission: "memories:write", projectId: memory.project_id }).state === "live";
  const wsPath = useWorkspacePath();
  const folder = memoryFolderOptions.find((o) => o.id === (memory.footer ? "footer" : "main"))?.name ?? "";
  const crumbs = useEmbeddedCrumbs([{ label: "Memories", to: wsPath("/memories") }, { label: folder }]);
  const interview = isInterviewMemory(memory);
  const interviewLength = useMemo(() => (interview ? bodyToMarkdown(body).length : 0), [interview, body]);

  const dirty = title !== memory.title || whenToUse !== memory.when_to_use || body !== memory.body;

  return (
    <div className="@container mx-auto w-full max-w-5xl">
      <PageHeader
        crumbs={crumbs}
        actionsOn="crumbs"
        className="border-b-0 pb-0"
        title={<DocTitleField editable={canWrite} title={title} staticTitle={memory.title} onChange={setTitle} />}
        meta={<MemorySettingsLine memory={memory} />}
        actions={
          <>
            {canWrite && (
              <Button size="sm" className="h-7" loading={saving} disabled={!dirty} onClick={() => onSave({ title, when_to_use: whenToUse, body })}>
                Save
              </Button>
            )}
            <MemoryActionsMenu memory={memory} onClone={() => setCloneOpen(true)} onDelete={onDelete} />
          </>
        }
      />
      {cloneOpen && <CloneMemoryDialog memoryId={memory.id} open onClose={() => setCloneOpen(false)} />}
      {/* The rail needs 50rem: a 35rem body, the gap and its 13rem; spare width sits between the body and the rail. */}
      <div className="mt-6 @min-[50rem]:flex @min-[50rem]:justify-between @min-[50rem]:gap-8">
        <article className="min-w-0 max-w-[42rem] flex-1 space-y-6">
          <MemoryWhenToUse value={canWrite ? whenToUse : memory.when_to_use} editable={canWrite} onChange={setWhenToUse} />
          {canWrite && <RichTextEditor value={body} onChange={setBody} aria-label="Body" attachTo={{ memory_id: memory.id }} />}
          {!canWrite && <DocBodyView body={memory.body} />}
          {interview && canWrite && interviewLength > 0 && (
            <div className="flex justify-end">
              <InterviewLengthMeter length={interviewLength} />
            </div>
          )}
          <div className="max-w-sm space-y-8 @min-[50rem]:hidden">
            <MemorySections memoryId={memory.id} currentVersion={memory.version} />
          </div>
        </article>
        <div className="hidden w-52 shrink-0 @min-[50rem]:block">
          <div className="sticky top-6 -mr-2.5 -ml-2 flex max-h-[calc(100vh-3rem)] flex-col gap-8 overflow-y-auto pl-2 [scrollbar-gutter:stable]">
            <MemorySections memoryId={memory.id} currentVersion={memory.version} />
          </div>
        </div>
      </div>
    </div>
  );
};

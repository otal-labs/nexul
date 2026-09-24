import { useMemo, useState } from "react";
import { Link } from "react-router";

import { AttachmentsSection } from "@/components/attachment/AttachmentsSection";
import { DocBodyView } from "@/components/doc/DocBodyView";
import { RichTextEditor } from "@/components/doc/RichTextEditor";
import { CloneMemoryDialog } from "@/components/memory/CloneMemoryDialog";
import { InterviewLengthMeter } from "@/components/memory/InterviewLengthMeter";
import { MemoryVersionsFeed } from "@/components/memory/MemoryVersionsFeed";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { isDecisionsLogMemory, isInterviewMemory, isWorkspaceMemory, type Memory } from "@/models/Memory";
import { bodyToMarkdown } from "@/utils/RichtextUtility";

interface MemoryDetailProps {
  memory: Memory;
  canWrite: boolean;
  canDelete: boolean;
  canClone: boolean;
  onSave: (input: { title: string; when_to_use: string; body: string; always_included: boolean }) => void;
  onDelete: () => void;
  saving: boolean;
}

// No live collaboration, unlike DocDetail: a memory is edited by one person at a time, saved explicitly.
export const MemoryDetail = ({ memory, canWrite, canDelete, canClone, onSave, onDelete, saving }: MemoryDetailProps) => {
  const [title, setTitle] = useState(memory.title);
  const [whenToUse, setWhenToUse] = useState(memory.when_to_use);
  const [alwaysIncluded, setAlwaysIncluded] = useState(memory.always_included);
  const [body, setBody] = useState(memory.body);
  const [cloneOpen, setCloneOpen] = useState(false);
  const interview = isInterviewMemory(memory);
  const decisionsLog = isDecisionsLogMemory(memory);
  const interviewLength = useMemo(() => (interview ? bodyToMarkdown(body).length : 0), [interview, body]);

  const dirty =
    title !== memory.title ||
    whenToUse !== memory.when_to_use ||
    alwaysIncluded !== memory.always_included ||
    body !== memory.body;

  return (
    <div className="animate-in fade-in-0 slide-in-from-bottom-1 mx-auto w-full max-w-6xl duration-200 ease-out">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Link
            to="/memories"
            className="font-mono text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground"
          >
            ← All memories
          </Link>
          {isWorkspaceMemory(memory) && (
            <span className="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
              Workspace
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
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
        </div>
      </div>
      {canClone && <CloneMemoryDialog memoryId={memory.id} open={cloneOpen} onClose={() => setCloneOpen(false)} />}

      <article className="relative mt-4 rounded-2xl border border-border bg-card p-6 shadow-card sm:p-10 lg:p-14">
        {canWrite && (
          <Input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            aria-label="Title"
            className="mb-4 border-none px-0 text-2xl font-semibold shadow-none focus-visible:ring-0"
          />
        )}
        {!canWrite && <h1 className="mb-4 text-2xl font-semibold">{memory.title}</h1>}

        {canWrite && (
          <div className="mb-6 space-y-3">
            <Input
              value={whenToUse}
              onChange={(e) => setWhenToUse(e.target.value)}
              placeholder="When should Agent use this? e.g. writing React code"
              aria-label="When to use"
              className="text-sm"
            />
            {!interview && !decisionsLog && (
              <label className="flex items-center gap-2 text-sm font-medium">
                <Switch checked={alwaysIncluded} onCheckedChange={setAlwaysIncluded} aria-label="Always included" />
                Always included in every turn
              </label>
            )}
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
              disabled={!dirty || saving}
              onClick={() => onSave({ title, when_to_use: whenToUse, body, always_included: alwaysIncluded })}
            >
              Save
            </Button>
          </div>
        )}

        <AttachmentsSection owner={{ memory_id: memory.id }} className="mt-8" />
      </article>

      <MemoryVersionsFeed memoryId={memory.id} currentVersion={memory.version} canRevert={canWrite} />
    </div>
  );
};

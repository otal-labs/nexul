import { CheckIcon, XIcon } from "lucide-react";
import { useState } from "react";

import { machineAnswer } from "@nexul/client-core/actionState";

import { ActionInlineEdit } from "@/components/action/ActionInlineEdit";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useUpdateMachine } from "@/hooks/RunnerHooks";
import type { Machine } from "@/models/Runner";

interface EditableStackRootProps {
  machine: Pick<Machine, "id" | "stack_root" | "actions">;
}

// editing/draft are genuine local UI state (F5); the mutation is the only source of the saved value.
export const EditableStackRoot = ({ machine }: EditableStackRootProps) => {
  const { id: machineId, stack_root: stackRoot } = machine;
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(stackRoot);
  const updateMachine = useUpdateMachine();

  const startEdit = () => {
    setDraft(stackRoot);
    setEditing(true);
  };

  const save = () => {
    const next = draft.trim();
    if (updateMachine.isPending) return;
    if (!next || next === stackRoot) {
      setEditing(false);
      return;
    }
    updateMachine.mutate({ id: machineId, stackRoot: next }, { onSuccess: () => setEditing(false) });
  };

  if (editing) {
    return (
      <form
        className="flex items-center gap-1"
        onSubmit={(e) => {
          e.preventDefault();
          save();
        }}
      >
        <Input
          autoFocus
          aria-label="Stack root"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={save}
          className="h-7 w-56 font-mono text-xs"
          disabled={updateMachine.isPending}
        />
        <Button
          type="submit"
          size="sm"
          variant="ghost"
          aria-label="Save stack root"
          loading={updateMachine.isPending}
          onMouseDown={(e) => e.preventDefault()}
        >
          <CheckIcon className="size-3.5" />
        </Button>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          aria-label="Cancel"
          onMouseDown={(e) => e.preventDefault()}
          onClick={() => setEditing(false)}
        >
          <XIcon className="size-3.5" />
        </Button>
      </form>
    );
  }

  return (
    <ActionInlineEdit
      permission="machines:write"
      answer={machineAnswer(machine, "machines:write")}
      label={`Edit stack root ${stackRoot}`}
      onEdit={startEdit}
      className="font-mono text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground"
    >
      {stackRoot}
    </ActionInlineEdit>
  );
};

import { CheckIcon, PencilIcon, XIcon } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useUpdateMachine } from "@/hooks/RunnerHooks";

interface EditableStackRootProps {
  machineId: string;
  stackRoot: string;
}

// editing/draft are genuine local UI state (F5); the mutation is the only source of the saved value.
export const EditableStackRoot = ({ machineId, stackRoot }: EditableStackRootProps) => {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(stackRoot);
  const updateMachine = useUpdateMachine();

  const startEdit = () => {
    setDraft(stackRoot);
    setEditing(true);
  };

  const save = async () => {
    const next = draft.trim();
    if (next && next !== stackRoot) await updateMachine.mutateAsync({ id: machineId, stackRoot: next });
    setEditing(false);
  };

  if (editing) {
    return (
      <form
        className="flex items-center gap-1"
        onSubmit={(e) => {
          e.preventDefault();
          void save();
        }}
      >
        <Input
          autoFocus
          aria-label="Stack root"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={() => void save()}
          className="h-7 w-56 font-mono text-xs"
          disabled={updateMachine.isPending}
        />
        <Button type="submit" size="sm" variant="ghost" aria-label="Save stack root" disabled={updateMachine.isPending}>
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
    <button
      type="button"
      onClick={startEdit}
      className="group inline-flex items-center gap-1.5 font-mono text-xs text-muted-foreground transition-colors duration-150 ease-standard hover:text-foreground"
    >
      {stackRoot}
      <PencilIcon className="size-3 opacity-0 transition-opacity duration-150 ease-standard group-hover:opacity-100" aria-hidden />
    </button>
  );
};

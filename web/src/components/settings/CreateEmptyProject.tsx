import { FolderPlus } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useCreateEmptyHarnessProject } from "@/hooks/PairingHooks";

interface CreateEmptyProjectProps {
  computerId: string;
  onCreated: (harnessProjectId: string) => void;
}

// Not a form: the picker sits inside one, and a form cannot nest.
export const CreateEmptyProject = ({ computerId, onCreated }: CreateEmptyProjectProps) => {
  const [name, setName] = useState("");
  const create = useCreateEmptyHarnessProject(computerId);
  const submit = () =>
    create.mutate(name.trim(), {
      onSuccess: (project) => {
        setName("");
        onCreated(project.id);
      },
    });

  return (
    <div className="space-y-2">
      <p className="text-xs text-muted-foreground">
        Nothing to clone here. An empty project makes a new folder beside the computer's other T3 projects.
      </p>
      <div className="flex flex-wrap gap-2">
        <Input
          aria-label="Folder name"
          placeholder="e.g. my-notes"
          className="h-8 w-48"
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => {
            if (e.key !== "Enter") return;
            e.preventDefault();
            if (name.trim() !== "") submit();
          }}
        />
        <Button type="button" variant="outline" size="sm" disabled={name.trim() === ""} loading={create.isPending} onClick={submit}>
          <FolderPlus />
          Create empty project
        </Button>
      </div>
    </div>
  );
};

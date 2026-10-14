import { UsersIcon } from "lucide-react";

import { peopleWord } from "@/models/ProjectPeople";

interface WorkspaceArchiveDetailsProps {
  people: number | undefined;
  projects: number | undefined;
}

// A signal in the archive confirmation, not a gate: what leaves the workspace's lists for everyone in it.
export const WorkspaceArchiveDetails = ({ people, projects }: WorkspaceArchiveDetailsProps) => (
  <div className="flex gap-3 border-y border-border py-3">
    <UsersIcon className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
    <div className="space-y-1 text-sm">
      {people !== undefined && (
        <p className="font-medium">
          {peopleWord(people)} {people === 1 ? "loses" : "lose"} this workspace.
        </p>
      )}
      {projects !== undefined && (
        <p className="text-muted-foreground">
          {projects} {projects === 1 ? "project" : "projects"}, with their tickets, docs, chat and plays, go with it.
        </p>
      )}
    </div>
  </div>
);

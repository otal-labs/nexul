import { UsersIcon } from "lucide-react";

import { peopleWord } from "@/models/ProjectPeople";

interface PeopleLoseProjectProps {
  count: number;
}

// A signal in the delete confirmation, not a gate: everyone in the project loses it.
export const PeopleLoseProject = ({ count }: PeopleLoseProjectProps) => (
  <div className="flex gap-3 border-y border-border py-3">
    <UsersIcon className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
    <p className="text-sm font-medium">
      {peopleWord(count)} {count === 1 ? "loses" : "lose"} this project.
    </p>
  </div>
);

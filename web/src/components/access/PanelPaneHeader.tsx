import type { ReactNode } from "react";

import { personLabel, type Person } from "@nexul/client-core/person";

import { PersonAvatar } from "@/components/PersonAvatar";

interface PanelPaneHeaderProps {
  person: Person;
  title: string;
  line: string;
  action?: ReactNode;
}

// The pane's head: the person's 32px mark, a title and a muted line, an optional action on the right.
export const PanelPaneHeader = ({ person, title, line, action }: PanelPaneHeaderProps) => (
  <header className="flex items-center gap-3">
    <PersonAvatar login={person.login} src={person.avatar_url} className="size-8" label={personLabel(person)} />
    <div className="min-w-0 flex-1">
      <h3 className="truncate text-[15px] font-semibold tracking-tight">{title}</h3>
      <p className="truncate text-xs text-muted-foreground">{line}</p>
    </div>
    {action}
  </header>
);

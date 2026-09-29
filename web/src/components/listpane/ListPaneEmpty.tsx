import type { ComponentType } from "react";
import { PlusIcon } from "lucide-react";

import { EmptyState } from "@/components/EmptyState";
import { Button } from "@/components/ui/button";

interface ListPaneEmptyProps {
  icon: ComponentType<{ className?: string }>;
  message: string;
  onNew?: (() => void) | undefined;
}

export const ListPaneEmpty = ({ icon, message, onNew }: ListPaneEmptyProps) => (
  <EmptyState
    role="status"
    icon={icon}
    title={message}
    size="compact"
    className="border-0"
    action={
      onNew && (
        <Button variant="ghost" size="sm" onClick={onNew}>
          <PlusIcon className="size-3.5" aria-hidden />
          New
        </Button>
      )
    }
  />
);

interface ListPaneNoMatchProps {
  onClear: () => void;
}

export const ListPaneNoMatch = ({ onClear }: ListPaneNoMatchProps) => (
  <EmptyState
    role="status"
    title="Nothing matches"
    size="compact"
    className="border-0"
    action={
      <Button variant="ghost" size="sm" onClick={onClear}>
        Clear search
      </Button>
    }
  />
);

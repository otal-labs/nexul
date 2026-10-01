import { Button } from "@/components/ui/button";
import { TeamAccountActions } from "@/components/team/TeamAccountActions";
import type { TeamPerson } from "@/models/Team";

interface TeamDialogFooterProps {
  person: TeamPerson;
  failure: string | undefined;
  canConfirm: boolean;
  applying: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}

// Account actions on the left apply at once; Cancel and Confirm settle the held workspace changes.
export const TeamDialogFooter = ({ person, failure, canConfirm, applying, onCancel, onConfirm }: TeamDialogFooterProps) => (
  <div className="space-y-2 border-t border-border px-6 py-3">
    {failure && (
      <p role="alert" className="text-sm text-destructive">
        {failure}. The rest is still waiting for Confirm.
      </p>
    )}
    <div className="flex flex-wrap items-center gap-2">
      <TeamAccountActions person={person} />
      <div className="ml-auto flex items-center gap-2">
        <Button type="button" variant="ghost" size="sm" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="button" size="sm" disabled={!canConfirm} loading={applying} onClick={onConfirm}>
          Confirm
        </Button>
      </div>
    </div>
  </div>
);

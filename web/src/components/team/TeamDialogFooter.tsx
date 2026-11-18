import { Button } from "@/components/ui/button";
import { TeamAccountActions } from "@/components/team/TeamAccountActions";
import type { TeamPerson } from "@/models/Team";

interface TeamDialogFooterProps {
  person: TeamPerson;
  onClose: () => void;
}

// Account actions on the left apply at once; nothing else here changes anything.
export const TeamDialogFooter = ({ person, onClose }: TeamDialogFooterProps) => (
  <div className="flex flex-wrap items-center gap-2 border-t border-border px-6 py-3">
    <TeamAccountActions person={person} />
    <Button type="button" variant="ghost" size="sm" className="ml-auto" onClick={onClose}>
      Close
    </Button>
  </div>
);

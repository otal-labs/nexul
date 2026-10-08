import { Button } from "@/components/ui/button";

interface InvitationAcceptancePanelProps {
  pending: boolean;
  onAccept: () => void;
  onDecline: () => void;
}

export const InvitationAcceptancePanel = ({ pending, onAccept, onDecline }: InvitationAcceptancePanelProps) => (
  <div className="space-y-3 rounded-md border bg-card p-4">
    <p className="text-sm font-medium">You'll join these workspaces with the roles above.</p>
    <p className="text-xs text-muted-foreground">Permission overrides apply once you accept.</p>
    <div className="flex flex-col gap-2 sm:flex-row">
      <Button type="button" className="flex-1" loading={pending} onClick={onAccept}>Accept invitation</Button>
      <Button type="button" variant="outline" className="flex-1" onClick={onDecline}>Decline</Button>
    </div>
  </div>
);

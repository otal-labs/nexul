import { Button } from "@/components/ui/button";

interface InvitationAcceptancePanelProps {
  pending: boolean;
  onAccept: () => void;
  onDecline: () => void;
}

export const InvitationAcceptancePanel = ({ pending, onAccept, onDecline }: InvitationAcceptancePanelProps) => (
  <div className="space-y-3 rounded-md border bg-card p-4">
    <p className="text-sm font-medium">Accepting joins you to these workspaces with the roles and overrides above.</p>
    <div className="flex flex-col gap-2 sm:flex-row">
      <Button type="button" className="flex-1" loading={pending} onClick={onAccept}>Accept invitation</Button>
      <Button type="button" variant="outline" className="flex-1" onClick={onDecline}>Decline</Button>
    </div>
  </div>
);

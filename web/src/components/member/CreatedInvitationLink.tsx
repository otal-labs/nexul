import { useState } from "react";
import { Check, Copy } from "lucide-react";

import { Button } from "@/components/ui/button";
import type { CreatedInvitation } from "@/models/Invitation";

interface CreatedInvitationLinkProps {
  invitation: CreatedInvitation;
}

export const CreatedInvitationLink = ({ invitation }: CreatedInvitationLinkProps) => {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(invitation.url);
      setCopied(true);
    } catch {
      // The link remains visible for manual copying.
    }
  };

  return (
    <div className="space-y-3 rounded-md border bg-muted/30 p-4" role="status">
      <p className="text-sm font-medium">Copy this link now. It won&apos;t be shown again.</p>
      <p className="break-all rounded-md bg-card p-2 font-mono text-xs">{invitation.url}</p>
      <Button type="button" variant="outline" size="sm" onClick={() => void copy()}>{copied ? <Check className="size-4" /> : <Copy className="size-4" />}{copied ? "Copied" : "Copy link"}</Button>
    </div>
  );
};

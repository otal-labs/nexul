import type { ReactNode } from "react";

import { SignedOutHeader } from "@/components/auth/SignedOutHeader";
import { invitationTitle, type InvitationPreview } from "@/models/Invitation";
import { formatClockTime, formatShortDate } from "@/utils/TimeUtility";

interface InvitationConsentPanelProps {
  details?: InvitationPreview | undefined;
  children: ReactNode;
}

// The invitation itself, before sign-in and after: what the link opens, who is asking to join, and how long it works.
export const InvitationConsentPanel = ({ details, children }: InvitationConsentPanelProps) => (
  <>
    <SignedOutHeader title={invitationTitle(details)} subtitle="Docs, tickets, chat and deploys, on Nexul." />
    {children}
    {details && (
      <p className="mt-3 border-t border-border pt-3 font-mono text-xs text-muted-foreground">
        Works once · expires {formatShortDate(details.expires_at)}, {formatClockTime(details.expires_at)}
      </p>
    )}
  </>
);

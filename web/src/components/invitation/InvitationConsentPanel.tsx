import type { ReactNode } from "react";

import { InvitationHeader } from "@/components/invitation/InvitationHeader";
import { InvitationPanel } from "@/components/invitation/InvitationPanel";
import type { InvitationPreview, RefusedPerson } from "@/models/Invitation";
import { formatClockTime, formatShortDate } from "@/utils/TimeUtility";

interface InvitationConsentPanelProps {
  details?: InvitationPreview | undefined;
  person?: RefusedPerson | undefined;
  children: ReactNode;
}

// The invitation itself, before sign-in and after: who is asking, what the link gives, and how long it works.
export const InvitationConsentPanel = ({ details, person, children }: InvitationConsentPanelProps) => {
  // The page is served from the instance, so its own host names it until the link has been read.
  const host = window.location.host;
  return (
    <InvitationPanel>
      <InvitationHeader instanceName={details?.instance_name || host} host={host} person={person} />
      {children}
      {details && (
        <p className="mt-5 text-center font-mono text-[11px] text-muted-foreground">
          Works once · expires {formatShortDate(details.expires_at)}, {formatClockTime(details.expires_at)}
        </p>
      )}
    </InvitationPanel>
  );
};

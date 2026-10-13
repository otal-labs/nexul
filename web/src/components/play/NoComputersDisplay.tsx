import { EmptyState } from "@/components/EmptyState";
import { FixLink } from "@/components/FixLink";
import { ADD_COMPUTER_LINK, HARNESS_READINESS_COPY } from "@/models/Pairing";

// A person with no computer can't pick one; anyone signed in may add theirs.
export const NoComputersDisplay = () => (
  <EmptyState role="status" size="compact" title={HARNESS_READINESS_COPY.unpaired.message} action={<FixLink {...ADD_COMPUTER_LINK} />} />
);

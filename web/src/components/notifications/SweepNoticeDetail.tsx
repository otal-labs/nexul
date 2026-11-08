import { Container } from "@/components/Container";
import { SweepDigestView } from "@/components/sweep/SweepDigestView";

// A sweep's notice opens its digest in the inbox pane.
export const SweepNoticeDetail = ({ sweepId }: { sweepId: string }) => (
  <Container className="py-6">
    <SweepDigestView sweepId={sweepId} />
  </Container>
);

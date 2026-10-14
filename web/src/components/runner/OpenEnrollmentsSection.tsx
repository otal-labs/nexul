import { EnterList } from "@/components/EnterList";
import { Microheader } from "@/components/Microheader";
import { OpenEnrollmentRow } from "@/components/runner/OpenEnrollmentRow";
import type { OpenEnrollment } from "@/models/Runner";

interface OpenEnrollmentsSectionProps {
  heading: string;
  codes: OpenEnrollment[] | undefined;
  labelOf: (code: OpenEnrollment) => string;
}

// Absent while there are none: an unused code is rare, and the Add dialogs show a fresh one.
export const OpenEnrollmentsSection = ({ heading, codes, labelOf }: OpenEnrollmentsSectionProps) => {
  if (!codes || codes.length === 0) return null;
  return (
    <section className="space-y-2">
      <Microheader>{heading}</Microheader>
      <EnterList className="divide-y divide-border overflow-hidden rounded-md border">
        {codes.map((code) => (
          <OpenEnrollmentRow key={code.id} code={code} label={labelOf(code)} />
        ))}
      </EnterList>
    </section>
  );
};

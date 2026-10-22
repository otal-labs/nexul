import { HarnessProviderMark } from "@/components/model/HarnessProviderMark";
import type { ProviderSetupLine } from "@/models/Pairing";

interface SetupProviderMarksProps {
  lines: ProviderSetupLine[];
}

// The providers a setup status is about, as their marks; each name stays for hover and screen readers.
export const SetupProviderMarks = ({ lines }: SetupProviderMarksProps) => (
  <span className="inline-flex items-center gap-1.5 align-[-2px]">
    {lines.map((line) => (
      <span key={line.provider} title={line.name}>
        <HarnessProviderMark driver={line.provider} className="size-3.5" />
        <span className="sr-only">{line.name}</span>
      </span>
    ))}
  </span>
);

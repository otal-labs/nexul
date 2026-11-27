import { HarnessProviderMark } from "@/components/model/HarnessProviderMark";
import type { ProviderSetupLine } from "@/models/Pairing";

interface SetupProviderMarksProps {
  lines: ProviderSetupLine[];
  // What hovering a mark and a screen reader say; the provider's name by default.
  titleOf?: (line: ProviderSetupLine) => string;
}

// The providers a setup status is about, as their marks; each name stays for hover and screen readers.
export const SetupProviderMarks = ({ lines, titleOf = (line) => line.name }: SetupProviderMarksProps) => (
  <span className="inline-flex items-center gap-1.5 align-[-2px]">
    {lines.map((line) => (
      <span key={line.provider} title={titleOf(line)}>
        <HarnessProviderMark driver={line.provider} className="size-4" />
        <span className="sr-only">{titleOf(line)}</span>
      </span>
    ))}
  </span>
);

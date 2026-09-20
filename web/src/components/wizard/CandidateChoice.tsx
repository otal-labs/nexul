import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import type { Candidate } from "@/models/Repository";

const candidateKey = (c: Candidate) => `${c.kind}:${c.path}`;

interface CandidateRowProps {
  candidate: Candidate;
}

const CandidateRow = ({ candidate }: CandidateRowProps) => {
  const id = `candidate-${candidateKey(candidate)}`;
  return (
    <label
      htmlFor={id}
      className="-mx-2 flex cursor-pointer items-start gap-3 rounded-md px-2 py-3 transition-colors duration-150 ease-standard hover:bg-accent/40"
    >
      <RadioGroupItem id={id} value={candidateKey(candidate)} className="mt-0.5" />
      <span className="min-w-0 flex-1">
        <span className="flex items-center gap-2">
          <span className="text-sm font-medium capitalize">{candidate.kind}</span>
          <span className="font-mono text-xs text-muted-foreground">{candidate.path}</span>
        </span>
        {candidate.services.length > 0 && (
          <span className="mt-1 block text-xs text-muted-foreground">
            {candidate.services.map((s) => s.name + (s.ports.length ? `:${s.ports.join(",")}` : "")).join(" · ")}
          </span>
        )}
      </span>
    </label>
  );
};

interface CandidateChoiceProps {
  candidates: Candidate[];
  value: Candidate;
  onChange: (candidate: Candidate) => void;
}

// A monorepo scan can return several candidates (spec §5); compose is preselected by the repository step, this
// just lets the owner switch. A single-candidate scan skips the picker entirely (WizardServiceStep).
export const CandidateChoice = ({ candidates, value, onChange }: CandidateChoiceProps) => (
  <RadioGroup
    aria-label="Candidate"
    value={candidateKey(value)}
    onValueChange={(key) => {
      const next = candidates.find((c) => candidateKey(c) === key);
      if (next) onChange(next);
    }}
    className="gap-0 divide-y divide-border border-y border-border"
  >
    {candidates.map((candidate) => (
      <CandidateRow key={candidateKey(candidate)} candidate={candidate} />
    ))}
  </RadioGroup>
);

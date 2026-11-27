import { Fragment, type ReactNode } from "react";

import type { PassOver } from "@/models/Pairing";

const WHY: Partial<Record<PassOver["why"], string>> = {
  offline: "offline",
  expired_token: "pairing expired",
  setup_required: "setup needed",
  no_model: "no model",
  no_checkout: "no checkout",
  ambiguous_checkout: "several checkouts",
  clone_failed: "clone failed",
  no_chat_folder: "no chat folder",
  full: "full",
  no_plays: "takes no plays",
  no_auto_plays: "takes no auto plays",
};

const Host = ({ children }: { children: ReactNode }) => <span className="font-mono font-medium">{children}</span>;

const joined = (names: string[]) =>
  names.map((name, i) => (
    <Fragment key={name}>
      {i > 0 && (i === names.length - 1 ? " and " : ", ")}
      <Host>{name}</Host>
    </Fragment>
  ));

interface PlacementSummaryProps {
  count: number;
  computer: string;
  passed: PassOver[];
  // Computers of yours the project's own list leaves out.
  outside?: string[];
}

const clause = (names: string[], one: string, many = one) => (
  <>
    {joined(names)} {names.length > 1 ? "are" : "is"} {names.length > 1 ? many : one}
  </>
);

// Every play landing on one computer: "All 11 plays go to mini. mint and xps are offline; desk is not in this project's computers."
export const PlacementSummary = ({ count, computer, passed, outside = [] }: PlacementSummaryProps) => {
  const entries = [...new Map(passed.map((p) => [`${p.computer_id}:${p.why}`, p])).values()];
  const byReason = new Map<string, string[]>();
  for (const p of entries) byReason.set(WHY[p.why] ?? p.text, [...(byReason.get(WHY[p.why] ?? p.text) ?? []), p.name]);
  const parts = [...byReason.entries()].map(([reason, names]) => <Fragment key={reason}>{clause(names, reason)}</Fragment>);
  if (outside.length > 0) parts.push(<Fragment key="outside">{clause(outside, "not in this project's computers")}</Fragment>);

  return (
    <p className="text-sm text-pretty">
      {count === 1 ? "The play goes" : `All ${count} plays go`} to <Host>{computer}</Host>.
      {parts.length > 0 && (
        <>
          {" "}
          {parts.map((part, i) => (
            <Fragment key={i}>
              {i > 0 && "; "}
              {part}
            </Fragment>
          ))}
          .
        </>
      )}
    </p>
  );
};

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
}

// Every play landing on one computer is two sentences: "All 11 plays go to mini. mint and xps are skipped while offline."
export const PlacementSummary = ({ count, computer, passed }: PlacementSummaryProps) => {
  const entries = [...new Map(passed.map((p) => [`${p.computer_id}:${p.why}`, p])).values()];
  const reasons = new Set(entries.map((p) => WHY[p.why] ?? p.text));
  const names = [...new Set(entries.map((p) => p.name))];

  return (
    <p className="text-sm text-pretty">
      {count === 1 ? "The play goes" : `All ${count} plays go`} to <Host>{computer}</Host>.
      {entries.length > 0 && reasons.size === 1 && (
        <>
          {" "}
          {joined(names)} {names.length > 1 ? "are" : "is"} skipped{[...reasons][0] === "offline" ? " while offline" : `: ${[...reasons][0]}`}.
        </>
      )}
      {entries.length > 0 && reasons.size > 1 && (
        <>
          {" "}
          Skipped: {entries.map((p, i) => (
            <Fragment key={`${p.computer_id}:${p.why}`}>
              {i > 0 && ", "}
              <Host>{p.name}</Host> ({WHY[p.why] ?? p.text})
            </Fragment>
          ))}
          .
        </>
      )}
    </p>
  );
};

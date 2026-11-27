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

interface PlacementSummaryProps {
  count: number;
  computer: string;
  passed: PassOver[];
}

// Every play landing on one computer is one sentence: "All 11 plays go to mini; mint and xps are passed over: offline."
export const PlacementSummary = ({ count, computer, passed }: PlacementSummaryProps) => {
  const seen = new Map(passed.map((p) => [`${p.computer_id}:${p.why}`, p]));
  const entries = [...seen.values()];
  const reasons = new Set(entries.map((p) => WHY[p.why] ?? p.text));
  const names = entries.map((p) => p.name);
  const list = names.length > 1 ? `${names.slice(0, -1).join(", ")} and ${names[names.length - 1]}` : (names[0] ?? "");
  const passedText =
    reasons.size === 1
      ? `${list} ${names.length > 1 ? "are" : "is"} passed over: ${[...reasons][0]}`
      : `passed over: ${entries.map((p) => `${p.name} (${WHY[p.why] ?? p.text})`).join(", ")}`;

  return (
    <p className="rounded-md border border-border px-3 py-2.5 text-sm text-pretty">
      {count === 1 ? "The play goes" : `All ${count} plays go`} to <span className="font-mono font-medium">{computer}</span>
      {entries.length > 0 && `; ${passedText}`}.
    </p>
  );
};

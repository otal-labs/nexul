import { useShallow } from "zustand/react/shallow";

import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { ClarifyPrototypeArticle } from "@/components/doc/prototype/ClarifyPrototypeArticle";
import { ClarifyPrototypeHeader } from "@/components/doc/prototype/ClarifyPrototypeHeader";
import { ClarifyPrototypePanel } from "@/components/doc/prototype/ClarifyPrototypePanel";
import { pendingCount, useClarifyPrototypeStore } from "@/components/doc/prototype/ClarifyPrototypeStore";
import type { Doc } from "@/models/Doc";

const ViewSwitch = () => {
  const { tab, setTab, pending } = useClarifyPrototypeStore(
    useShallow((s) => ({ tab: s.tab, setTab: s.setTab, pending: pendingCount(s) })),
  );
  return (
    <ToggleGroup
      type="single"
      variant="outline"
      size="sm"
      value={tab}
      onValueChange={(value) => value && setTab(value as "doc" | "questions")}
      aria-label="Doc or questions"
    >
      <ToggleGroupItem value="doc">Doc</ToggleGroupItem>
      <ToggleGroupItem value="questions" className="gap-1.5">
        Questions
        {pending > 0 && <span className="font-mono text-xs text-muted-foreground tabular-nums">{pending}</span>}
      </ToggleGroupItem>
    </ToggleGroup>
  );
};

// C: a Doc | Questions switch in the header; the Questions view takes the article's place at full width.
export const ClarifyPrototypeVariantC = ({ doc }: { doc: Doc }) => {
  const tab = useClarifyPrototypeStore((s) => s.tab);
  return (
    <div className="mx-auto w-full max-w-4xl">
      <ClarifyPrototypeHeader doc={doc} start={<ViewSwitch />} />
      {tab === "doc" && <ClarifyPrototypeArticle title={doc.title} className="mt-4" />}
      {tab === "questions" && (
        <div className="mt-4 rounded-2xl border border-border bg-card p-6 shadow-card sm:p-10">
          <ClarifyPrototypePanel className="mx-auto max-w-2xl" />
        </div>
      )}
    </div>
  );
};

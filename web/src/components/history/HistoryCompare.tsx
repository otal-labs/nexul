import type { Ref } from "react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { HistoryCodeDiff } from "@/components/history/HistoryCodeDiff";
import { HistoryDocument } from "@/components/history/HistoryDocument";
import { HistoryScrollFade } from "@/components/history/HistoryScrollFade";
import { HistoryStrip } from "@/components/history/HistoryStrip";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import type { HistoryCompare as Compare } from "@/hooks/HistoryCompareHooks";
import { cn } from "@/lib/utils";
import type { HistoryBodyKind, HistoryVersionSummary, HistoryView } from "@/models/History";

interface HistoryCompareProps {
  ref: Ref<HTMLDivElement>;
  picked: HistoryVersionSummary;
  /** "Restored from 24 Oct 16:24" when the picked version is a restore, else empty. */
  restored: string;
  previous: HistoryVersionSummary | undefined;
  compare: Compare;
  view: HistoryView;
  onView: (view: HistoryView) => void;
  step: number;
  onHop: (to: number) => void;
  bodyKind: HistoryBodyKind;
}

// The body column: the compare strip over the picked version's text, which scrolls on its own below the strip's hairline, its
// scrollbar kept clear of the rail's. A change lands 40px down, clear of the top fade, so a hop never ends under it. The body's
// scrollbar gutter is reserved, so its text never moves between states.
export const HistoryCompare = ({ ref, picked, restored, previous, compare, view, onView, step, onHop, bodyKind }: HistoryCompareProps) => (
  <div className="@container flex min-h-0 min-w-0 flex-col bg-popover">
    <HistoryStrip
      restored={restored}
      picked={picked}
      previous={previous}
      view={view}
      onView={onView}
      ready={!!compare.runs}
      changes={compare.changes}
      step={step}
      onHop={onHop}
    />
    <div ref={ref} className="mr-2 min-h-0 flex-1 overflow-y-auto [scroll-padding-top:40px] [scrollbar-gutter:stable]">
      <HistoryScrollFade edge="top" size="lg" flush />
      <div className={cn("pt-8 pb-4", bodyKind === "code" ? "px-6" : "px-10 @min-[34rem]:px-14")}>
        {compare.isPending && !compare.error && <LoadingDisplay />}
        {compare.error && <ErrorDisplay error={compare.error} title="Couldn't load this version." />}
        {compare.runs && bodyKind === "code" && <HistoryCodeDiff key={`${picked.version}-${view}`} runs={compare.runs} changes={compare.changes} />}
        {compare.runs && bodyKind !== "code" && <HistoryDocument title={compare.title} titleChanged={compare.titleChanged} runs={compare.runs} changes={compare.changes} />}
      </div>
      <HistoryScrollFade edge="bottom" />
    </div>
  </div>
);

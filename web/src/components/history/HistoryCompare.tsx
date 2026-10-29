import type { Ref } from "react";

import { ErrorDisplay } from "@/components/ErrorDisplay";
import { HistoryDocument } from "@/components/history/HistoryDocument";
import { HistoryScrollFade } from "@/components/history/HistoryScrollFade";
import { HistoryStrip } from "@/components/history/HistoryStrip";
import { LoadingDisplay } from "@/components/LoadingDisplay";
import type { HistoryCompare as Compare } from "@/hooks/HistoryCompareHooks";
import type { HistoryVersionSummary, HistoryView } from "@/models/History";

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
}

// The body column: the compare strip over the picked version's text, which scrolls on its own below the strip's hairline, its
// scrollbar kept clear of the rail's. A change lands 40px down, clear of the top fade, so a hop never ends under it. The body's
// scrollbar gutter is reserved, so its text never moves between states.
export const HistoryCompare = ({ ref, picked, restored, previous, compare, view, onView, step, onHop }: HistoryCompareProps) => (
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
      <div className="px-10 pt-8 pb-4 @min-[34rem]:px-14">
        {compare.isPending && !compare.error && <LoadingDisplay />}
        {compare.error && <ErrorDisplay error={compare.error} title="Couldn't load this version." />}
        {compare.runs && <HistoryDocument title={compare.title} titleChanged={compare.titleChanged} runs={compare.runs} changes={compare.changes} />}
      </div>
      <HistoryScrollFade edge="bottom" />
    </div>
  </div>
);

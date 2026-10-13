import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router";

import { pageTitleClassFor } from "@/components/PageHeader";
import { SettingsCard } from "@/components/settings/SettingsCard";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

import "./picker.css";
import { InvitationsTab, PersonDialog } from "./shared";
import { ColumnVariant, GapLine, GapLineVariant, StripVariant } from "./variants";

const variants = [
  { name: "Gap line", View: GapLineVariant },
  { name: "Progress column", View: ColumnVariant },
  { name: "Step strip", View: StripVariant },
];

// Throwaway: /proto/team-progress?v=1&person=bob&tab=invitations
export const TeamProgressProto = () => {
  const [params, setParams] = useSearchParams();
  const current = Math.min(Math.max((Number(params.get("v")) || 1) - 1, 0), variants.length - 1);
  const personId = params.get("person");
  const tab = params.get("tab") ?? "people";
  const picker = useRef<HTMLElement>(null);
  const highlight = useRef<HTMLSpanElement>(null);
  const [ready, setReady] = useState(false);

  const set = (key: string, value: string | null) =>
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        if (value === null) next.delete(key);
        if (value !== null) next.set(key, value);
        return next;
      },
      { replace: true },
    );

  useLayoutEffect(() => {
    const items = picker.current?.querySelectorAll<HTMLButtonElement>(".proto-picker-item");
    const el = items?.[current];
    if (!el || !highlight.current) return;
    highlight.current.style.width = `${el.offsetWidth}px`;
    highlight.current.style.transform = `translateX(${el.offsetLeft}px)`;
  }, [current]);

  useEffect(() => {
    requestAnimationFrame(() => requestAnimationFrame(() => setReady(true)));
    const onKey = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      if (/^(INPUT|TEXTAREA|SELECT)$/.test(target.tagName) || target.isContentEditable) return;
      if (e.metaKey || e.ctrlKey || e.altKey) return;
      const num = parseInt(e.key, 10);
      if (num >= 1 && num <= variants.length) set("v", String(num));
      if (e.key === "ArrowRight") set("v", String(((current + 1) % variants.length) + 1));
      if (e.key === "ArrowLeft") set("v", String(((current - 1 + variants.length) % variants.length) + 1));
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  });

  const View = variants[current]!.View;
  const open = (id: string | null) => set("person", id);

  return (
    <div className="flex min-h-dvh">
      <div className="light-field" aria-hidden />
      <div aria-hidden className="w-14 shrink-0 lg:w-60" />
      <main className="min-w-0 flex-1 p-2">
        <div className="panel min-h-[calc(100dvh-1rem)] rounded-xl">
          <div className="mx-auto w-full max-w-5xl space-y-6 px-4 py-8 sm:px-6">
            <header className="border-b border-border pb-5">
              <h1 className={pageTitleClassFor("Configuration")}>Configuration</h1>
              <p className="mt-1 text-sm text-muted-foreground">Acme · Team</p>
            </header>
            <SettingsCard
              id="team"
              title="Team"
              description="Everyone on this instance, and what they can reach in each workspace. Invite someone with a link."
              footer={<Button variant="outline">Invite people</Button>}
            >
              <Tabs value={tab} onValueChange={(value) => set("tab", value === "people" ? null : value)} className="gap-6">
                <TabsList variant="line" className="w-full justify-start border-b border-border p-0">
                  <TabsTrigger value="people" className="flex-none px-3">
                    People
                  </TabsTrigger>
                  <TabsTrigger value="invitations" className="flex-none px-3">
                    Invitations
                  </TabsTrigger>
                </TabsList>
                <TabsContent value="people">
                  <View onOpen={open} />
                </TabsContent>
                <TabsContent value="invitations">
                  <InvitationsTab onOpen={open} renderProgress={(person) => <GapLine person={person} />} />
                </TabsContent>
              </Tabs>
            </SettingsCard>
          </div>
        </div>
      </main>
      <PersonDialog personId={personId} onClose={() => open(null)} />
      <nav ref={picker} className="proto-picker" aria-label="Prototype variants" data-ready={ready ? "" : undefined}>
        <span ref={highlight} className="proto-picker-highlight" aria-hidden="true" />
        {variants.map((variant, i) => (
          <button
            key={variant.name}
            type="button"
            className="proto-picker-item"
            data-active={i === current ? "" : undefined}
            aria-current={i === current ? "true" : undefined}
            onClick={() => set("v", String(i + 1))}
          >
            {variant.name}
          </button>
        ))}
      </nav>
    </div>
  );
};

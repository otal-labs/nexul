import { ChevronRight } from "lucide-react";
import type { ReactNode } from "react";

import { Logo } from "@/components/Logo";
import { microheaderClass } from "@/components/Microheader";
import { displayTitleClass } from "@/components/PageHeader";
import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import { FlowContext, type Flow } from "./flow";
import { doneCount, STEPS, useWelcomeProto, type StepKey } from "./model";
import { FinishView, STATE_WORD, StepBody, StepNode } from "./parts";

export interface FrameProps {
  current: StepKey | "finish";
  flow: Flow | null;
  owner: boolean;
  open: (step: StepKey) => void;
  forward: boolean;
}

const TopBar = () => (
  <header className="flex items-center justify-between">
    <Logo className="size-7" />
    <Button variant="ghost" size="sm" className="text-muted-foreground">
      Finish later
    </Button>
  </header>
);

const StepHead = ({ step }: { step: StepKey }) => (
  <>
    <h1 className={cn(displayTitleClass, "text-[2rem]")}>{STEPS[step].title}</h1>
    <p className="mt-2 text-muted-foreground">{STEPS[step].lead}</p>
  </>
);

// The step slides 8px in from the side it was reached from, the project wizard's 180ms.
const Slide = ({ id, forward, children }: { id: string; forward: boolean; children: ReactNode }) => (
  <div
    key={id}
    className={cn(
      "animate-in fade-in-0 duration-180 ease-out motion-reduce:animate-none",
      forward ? "slide-in-from-right-2" : "slide-in-from-left-2",
    )}
  >
    {children}
  </div>
);

const Progress = () => {
  const order = useWelcomeProto((s) => s.order);
  const states = useWelcomeProto((s) => s.states);
  return (
    <p className="font-mono text-xs text-muted-foreground tabular-nums">
      {doneCount(order, states)} of {order.length} done
    </p>
  );
};

// Rail: the step list holds its own column beside the step for the whole wizard.
const RailList = ({ current, open }: Pick<FrameProps, "current" | "open">) => {
  const order = useWelcomeProto((s) => s.order);
  const states = useWelcomeProto((s) => s.states);
  return (
    <nav aria-label="Welcome steps" className="@2xl:border-r @2xl:border-border @2xl:pr-8">
      <p className={microheaderClass}>Welcome</p>
      <div className="mt-1">
        <Progress />
      </div>
      <ol className="mt-6">
        {order.map((k, i) => {
          const isCurrent = k === current;
          const state = isCurrent ? "current" : states[k];
          return (
            <li key={k} className="relative">
              {i < order.length - 1 && <span aria-hidden className="absolute top-7 bottom-0 left-[9.5px] w-px bg-border" />}
              <button
                type="button"
                aria-current={isCurrent ? "step" : undefined}
                onClick={() => open(k)}
                className="quiet-focus group flex w-full items-start gap-3 rounded-md py-2 text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
              >
                <StepNode state={state} />
                <span className="flex min-w-0 flex-col">
                  <span
                    className={cn(
                      "text-sm leading-5 transition-colors duration-150 ease-standard",
                      isCurrent ? "font-medium text-foreground" : "text-muted-foreground group-hover:text-foreground",
                    )}
                  >
                    {STEPS[k].name}
                  </span>
                  {!isCurrent && states[k] !== "todo" && (
                    <span className="font-mono text-[11px] text-muted-foreground/80">{STATE_WORD[states[k]]}</span>
                  )}
                </span>
              </button>
            </li>
          );
        })}
      </ol>
    </nav>
  );
};

export const RailFrame = ({ current, flow, owner, open, forward }: FrameProps) => (
  <ShowcaseSurface quiet>
    <div className="mx-auto flex min-h-full w-full max-w-6xl flex-col px-6 py-6 lg:px-10">
      <TopBar />
      <div className="@container mt-10 flex-1 pb-24 lg:mt-16">
        <div className="grid gap-10 @2xl:grid-cols-[13rem_minmax(0,1fr)] @2xl:gap-12 @4xl:grid-cols-[15rem_minmax(0,1fr)] @4xl:gap-16">
          <RailList current={current} open={open} />
          <main className="min-w-0 max-w-3xl">
            <Slide id={current} forward={forward}>
              {current === "finish" && <FinishView onOpen={open} owner={owner} />}
              {current !== "finish" && flow && (
                <FlowContext.Provider value={flow}>
                  <StepHead step={current} />
                  <div className="mt-8">
                    <StepBody step={current} />
                  </div>
                </FlowContext.Provider>
              )}
            </Slide>
          </main>
        </div>
      </div>
    </div>
  </ShowcaseSurface>
);

// Row: the project wizard's horizontal node row above a reading-width column.
const RowNodes = ({ current, open }: Pick<FrameProps, "current" | "open">) => {
  const order = useWelcomeProto((s) => s.order);
  const states = useWelcomeProto((s) => s.states);
  const at = current === "finish" ? order.length : order.indexOf(current);
  return (
    <div className="@container">
      <ol aria-label="Welcome steps" className="flex">
        {order.map((k, i) => (
          <li key={k} className="relative flex flex-1 flex-col items-center">
            {i > 0 && (
              <span aria-hidden className="absolute top-2.5 right-1/2 h-px w-full bg-border">
                <span className={cn("block h-full origin-left bg-brand transition-transform duration-250 ease-standard", i <= at ? "scale-x-100" : "scale-x-0")} />
              </span>
            )}
            <button type="button" onClick={() => open(k)} className="rounded-full">
              <StepNode state={k === current ? "current" : states[k]} />
            </button>
            <span className={cn("mt-2 hidden px-1 text-center font-mono text-xs leading-tight @2xl:block", k === current ? "font-bold text-foreground" : "text-muted-foreground")}>
              {STEPS[k].name}
            </span>
          </li>
        ))}
      </ol>
    </div>
  );
};

export const RowFrame = ({ current, flow, owner, open, forward }: FrameProps) => (
  <ShowcaseSurface quiet>
    <div className="mx-auto flex min-h-full w-full max-w-5xl flex-col px-6 py-6 lg:px-10">
      <TopBar />
      <div className="mx-auto mt-10 w-full max-w-3xl lg:mt-14">
        <RowNodes current={current} open={open} />
      </div>
      <main className="mx-auto mt-12 w-full max-w-xl pb-24">
        <Slide id={current} forward={forward}>
          {current === "finish" && <FinishView onOpen={open} owner={owner} />}
          {current !== "finish" && flow && (
            <FlowContext.Provider value={flow}>
              <StepHead step={current} />
              <div className="mt-8">
                <StepBody step={current} />
              </div>
            </FlowContext.Provider>
          )}
        </Slide>
      </main>
    </div>
  </ShowcaseSurface>
);

// Hub: one checklist; each row opens in place, one at a time, and Finish sits under the list.
const HubRow = ({ step, current, flow, open }: { step: StepKey } & Pick<FrameProps, "current" | "flow" | "open">) => {
  const state = useWelcomeProto((s) => s.states[step]);
  const isOpen = step === current;
  return (
    <li className={cn("transition-colors duration-150 ease-standard", isOpen && "bg-card/60")}>
      <button
        type="button"
        aria-expanded={isOpen}
        onClick={() => open(step)}
        className="flex w-full items-center gap-3 px-4 py-3.5 text-left hover:bg-accent/40"
      >
        <StepNode state={isOpen ? "current" : state} />
        <span className="min-w-0 flex-1">
          <span className="block text-sm font-medium">{STEPS[step].name}</span>
          <span className="block truncate text-xs text-muted-foreground">{STEPS[step].line}</span>
        </span>
        {state !== "todo" && <span className="font-mono text-xs text-muted-foreground">{STATE_WORD[state]}</span>}
        <ChevronRight aria-hidden className={cn("size-4 text-muted-foreground transition-transform duration-150 ease-standard", isOpen && "rotate-90")} />
      </button>
      {isOpen && flow && (
        <div className="animate-in fade-in-0 slide-in-from-top-1 px-4 pb-5 pl-12 duration-200 ease-out">
          <p className="mb-5 text-sm text-muted-foreground">{STEPS[step].lead}</p>
          <FlowContext.Provider value={{ ...flow, compact: true }}>
            <StepBody step={step} />
          </FlowContext.Provider>
        </div>
      )}
    </li>
  );
};

export const HubFrame = ({ current, flow, owner, open }: FrameProps) => {
  const order = useWelcomeProto((s) => s.order);
  const states = useWelcomeProto((s) => s.states);
  const pct = (doneCount(order, states) / order.length) * 100;
  return (
    <ShowcaseSurface quiet>
      <div className="mx-auto flex min-h-full w-full max-w-5xl flex-col px-6 py-6 lg:px-10">
        <TopBar />
        <main className="mx-auto mt-10 w-full max-w-2xl pb-24 lg:mt-14">
          <h1 className={cn(displayTitleClass, "text-[2rem]")}>Welcome, Alice</h1>
          <p className="mt-2 text-muted-foreground">Set up what's yours. Skip anything; it waits under Welcome in your account menu.</p>
          <div className="mt-6 flex items-center gap-3">
            <div className="h-1 flex-1 overflow-hidden rounded-full bg-border">
              <div className="h-full origin-left rounded-full bg-brand transition-transform duration-250 ease-standard" style={{ transform: `scaleX(${pct / 100})` }} />
            </div>
            <Progress />
          </div>
          <ol className="mt-6 divide-y divide-border overflow-hidden rounded-lg border border-border">
            {order.map((k) => (
              <HubRow key={k} step={k} current={current} flow={flow} open={open} />
            ))}
          </ol>
          <div className="flex justify-end pt-8">
            <Button>{owner ? "Create your first project" : "Go to Northwind"}</Button>
          </div>
        </main>
      </div>
    </ShowcaseSurface>
  );
};

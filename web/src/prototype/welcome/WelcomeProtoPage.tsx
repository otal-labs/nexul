import { useEffect, useState } from "react";

import { WizardConfirmation } from "@/components/auth/WizardConfirmation";
import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";

import type { Flow } from "./flow";
import { HubFrame, RailFrame, RowFrame, type FrameProps } from "./frames";
import { useWelcomeProto, type Scene, type StepKey } from "./model";
import { Picker } from "./Picker";

const VARIANTS = [
  { name: "Rail", Frame: RailFrame },
  { name: "Row", Frame: RowFrame },
  { name: "Hub", Frame: HubFrame },
];

const params = () => new URLSearchParams(location.search);

const setParam = (key: string, value: string) => {
  const url = new URL(location.href);
  url.searchParams.set(key, value);
  history.replaceState(null, "", url);
};

// Prototype harness: one variant at a time, full size, the picker bottom centre; ?scene, ?step and ?handover seed it.
export const WelcomeProtoPage = () => {
  const [variant, setVariant] = useState(() => Math.max(0, (Number(params().get("v")) || 1) - 1));
  const [mountKey, setMountKey] = useState(0);
  const load = useWelcomeProto((s) => s.load);
  const mark = useWelcomeProto((s) => s.mark);
  const order = useWelcomeProto((s) => s.order);
  const states = useWelcomeProto((s) => s.states);
  const scene = useWelcomeProto((s) => s.scene);
  const [ready, setReady] = useState(false);
  const [current, setCurrent] = useState<StepKey | "finish">("profile");
  const [forward, setForward] = useState(true);
  const [handover, setHandover] = useState(() => params().get("handover") === "1");

  useEffect(() => {
    const s = (params().get("scene") as Scene | null) ?? "member";
    load(s);
    const st = useWelcomeProto.getState();
    const first = (params().get("step") as StepKey | "finish" | null) ?? st.order.find((k) => st.states[k] === "todo") ?? "finish";
    setCurrent(first);
    setReady(true);
  }, [load]);

  useEffect(() => {
    if (!handover || params().get("freeze")) return;
    const t = setTimeout(() => setHandover(false), 1600);
    return () => clearTimeout(t);
  }, [handover]);

  const go = (to: StepKey | "finish") => {
    const idx = (k: StepKey | "finish") => (k === "finish" ? order.length : order.indexOf(k));
    setForward(idx(to) >= idx(current));
    setCurrent(to);
  };

  const flow: Flow | null =
    current === "finish"
      ? null
      : {
          step: current,
          first: order.indexOf(current) === 0,
          last: order.indexOf(current) === order.length - 1,
          back: () => go(order[order.indexOf(current) - 1] ?? current),
          next: (as) => {
            if (as === "skipped" || states[current] !== "done") mark(current, as);
            go(order[order.indexOf(current) + 1] ?? "finish");
          },
        };

  const pick = (i: number) => {
    setVariant(i);
    setParam("v", String(i + 1));
  };

  const { Frame } = VARIANTS[variant] ?? VARIANTS[0]!;
  const props: FrameProps = { current, flow, owner: scene === "owner", open: go, forward };

  return (
    <>
      {handover && (
        <ShowcaseSurface quiet>
          <div className="flex min-h-full items-center justify-center px-6">
            <WizardConfirmation title="Workspace ready" subtitle="Next, what's yours to set up: GitHub and your computer." />
          </div>
        </ShowcaseSurface>
      )}
      {!handover && ready && <Frame key={mountKey} {...props} />}
      <Picker names={VARIANTS.map((v) => v.name)} active={variant} onPick={pick} onReplay={() => setMountKey((k) => k + 1)} />
    </>
  );
};

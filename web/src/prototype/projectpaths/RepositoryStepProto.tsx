import { useState, type ComponentType } from "react";

import { WizardLayout } from "@/components/auth/WizardLayout";
import type { WizardProgressState } from "@/components/wizard/WizardProgressItem";
import type { WizardStepId } from "@/models/ProjectWizard";

import { DoneNextPlaces, DonePerStep, type Exit } from "./DoneStep";
import { DEPLOY_STEPS, repoStepStates, type Scene } from "./mock";
import type { RepoStepProps } from "./parts";
import { Picker } from "./Picker";
import { ProtoProgress } from "./ProtoProgress";
import { RepoStepChoice } from "./RepoStepChoice";
import { RepoStepFooter } from "./RepoStepFooter";
import { RepoStepRow } from "./RepoStepRow";

const REPO_VARIANTS: { name: string; Step: ComponentType<RepoStepProps> }[] = [
  { name: "Footer", Step: RepoStepFooter },
  { name: "On the row", Step: RepoStepRow },
  { name: "Three ways", Step: RepoStepChoice },
];

const DONE_VARIANTS = [
  { name: "Per step", Done: DonePerStep },
  { name: "Next places", Done: DoneNextPlaces },
  { name: "Next places, marked row", Done: DoneNextPlaces },
];

const params = () => new URLSearchParams(location.search);

const setParam = (key: string, value: string) => {
  const url = new URL(location.href);
  url.searchParams.set(key, value);
  history.replaceState(null, "", url);
};

const index = (key: string, max: number) => Math.min(max - 1, Math.max(0, (Number(params().get(key)) || 1) - 1));

// Variant 1 marks every deploy step skipped; variant 2 leaves them unmarked, faded, with the line broken over them.
const doneStates = (exit: Exit, marked: boolean): Record<WizardStepId, WizardProgressState> => {
  const deploy: WizardProgressState = marked ? "skipped" : "unvisited";
  return {
    project: "done",
    people: "done",
    repository: exit === "none" ? "skipped" : "done",
    service: deploy,
    env: deploy,
    reach: deploy,
    branches: deploy,
    done: "current",
  };
};

// Prototype harness for the Repository step's choices and the early Done: ?scene=search|picked|done-none|done-attached, ?v, ?d.
export const RepositoryStepProto = () => {
  const [scene, setSceneState] = useState<Scene>(() => (params().get("scene") as Scene | null) ?? "search");
  const [variant, setVariant] = useState(() => index("v", REPO_VARIANTS.length));
  const [doneVariant, setDoneVariant] = useState(() => index("d", DONE_VARIANTS.length));
  const [mountKey, setMountKey] = useState(0);
  const onDone = scene === "done-none" || scene === "done-attached";
  const exit: Exit = scene === "done-attached" ? "attached" : "none";

  const setScene = (next: Scene) => {
    setSceneState(next);
    setParam("scene", next);
  };
  const pick = (i: number) => {
    if (onDone) {
      setDoneVariant(i);
      setParam("d", String(i + 1));
      return;
    }
    setVariant(i);
    setParam("v", String(i + 1));
  };

  const { Step } = REPO_VARIANTS[variant]!;
  const { Done } = DONE_VARIANTS[doneVariant]!;
  // Variant 2 tries the unmarked, broken line; 1 and 3 record the deploy steps skipped as ADR 0143 draws them.
  const marked = doneVariant !== 1;
  const progress = onDone ? (
    <ProtoProgress
      states={doneStates(exit, marked)}
      gaps={marked ? [] : [...DEPLOY_STEPS, "done"]}
      faded={marked ? [] : DEPLOY_STEPS}
    />
  ) : (
    <ProtoProgress states={repoStepStates()} />
  );

  return (
    <>
      <WizardLayout progress={progress} title="New project" subtitle="Name the project, then pick its repository if it has one.">
        {!onDone && (
          <Step
            key={mountKey}
            picked={scene === "picked"}
            onPick={() => setScene("picked")}
            onUnpick={() => setScene("search")}
            onExit={(kind) => setScene(kind === "none" ? "done-none" : "done-attached")}
          />
        )}
        {onDone && (
          <div>
            <h2 className="text-base font-semibold tracking-tight">Done</h2>
            <div className="mt-4">
              <Done key={mountKey} exit={exit} onBack={() => setScene("search")} />
            </div>
          </div>
        )}
      </WizardLayout>
      <Picker
        names={(onDone ? DONE_VARIANTS : REPO_VARIANTS).map((v) => v.name)}
        active={onDone ? doneVariant : variant}
        onPick={pick}
        onReplay={() => setMountKey((k) => k + 1)}
      />
    </>
  );
};

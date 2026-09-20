import { useNavigate, useSearchParams } from "react-router";
import { useShallow } from "zustand/react/shallow";

import { DnsStep, type DnsStepState } from "@/components/dns/DnsStep";
import { WizardDoneStep } from "@/components/wizard/WizardDoneStep";
import { WizardEnvStep } from "@/components/wizard/WizardEnvStep";
import { WizardProjectStep } from "@/components/wizard/WizardProjectStep";
import { WizardReachStep } from "@/components/wizard/WizardReachStep";
import { WizardRepositoryStep } from "@/components/wizard/WizardRepositoryStep";
import { WizardServiceStep } from "@/components/wizard/WizardServiceStep";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

export const WizardSteps = ["project", "repository", "service", "env", "reach", "done"] as const;
export type WizardStepId = (typeof WizardSteps)[number];

interface ProjectWizardStepperProps {
  step: WizardStepId;
}

// The DNS onboarding rail (DnsStep/DnsSetupStepper), driven by the URL step instead of local state: a step's
// state is purely its position relative to the current one, so moving the URL forward is enough to collapse
// everything before it to "done" and light the next one — no separate completion flags to keep in sync.
export const ProjectWizardStepper = ({ step }: ProjectWizardStepperProps) => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const {
    projectName,
    projectPreselected,
    attachStackId,
    repository,
    scanResult,
    candidate,
    name,
    machine,
    exposureHostname,
    stackId,
  } = useProjectWizardStore(
    useShallow((s) => ({
      projectName: s.projectName,
      projectPreselected: s.projectPreselected,
      attachStackId: s.attachStackId,
      repository: s.repository,
      scanResult: s.scanResult,
      candidate: s.candidate,
      name: s.name,
      machine: s.machine,
      exposureHostname: s.exposureHostname,
      stackId: s.stackId,
    })),
  );

  const isAttach = !!attachStackId;
  const showEnv = (scanResult?.env_keys.length ?? 0) > 0;
  const order = WizardSteps.filter((id) => (id !== "env" || showEnv) && (id !== "reach" || !isAttach));
  const currentIndex = order.indexOf(step);
  // Attach mode has no name/machine to enter, so its summary names the candidate instead.
  const serviceSummary = (): string | undefined => {
    if (isAttach) return candidate ? `${candidate.kind === "compose" ? "Compose" : "Dockerfile"} · ${candidate.path}` : undefined;
    return machine ? `${name} on ${machine}` : name;
  };
  // Attach mode skips the Reach rung (the stack already has hostnames), so service and env land on "done".
  const afterService = (): WizardStepId => {
    if (showEnv) return "env";
    return isAttach ? "done" : "reach";
  };
  const afterEnv = isAttach ? "done" : "reach";

  const goTo = (next: WizardStepId) => {
    const query = searchParams.toString();
    navigate(`/wizard/project/${next}${query ? `?${query}` : ""}`);
  };

  const stateFor = (id: WizardStepId): DnsStepState => {
    const idx = order.indexOf(id);
    if (idx === currentIndex) return "active";
    return idx < currentIndex ? "done" : "upcoming";
  };

  return (
    <ol className="list-none">
      <DnsStep
        title="Project"
        state={stateFor("project")}
        summary={projectName}
        {...(!projectPreselected && { onChange: () => goTo("project") })}
      >
        <WizardProjectStep onDone={() => goTo("repository")} />
      </DnsStep>
      <DnsStep
        title="Repository"
        description="Pick the repository to deploy; Nexul reads its Dockerfile or compose file."
        state={stateFor("repository")}
        summary={repository?.full_name}
        onChange={() => goTo("repository")}
      >
        <WizardRepositoryStep onDone={() => goTo("service")} />
      </DnsStep>
      <DnsStep
        title={isAttach ? "Attach" : "Service"}
        state={stateFor("service")}
        summary={serviceSummary()}
        onChange={() => goTo("service")}
      >
        <WizardServiceStep onDone={() => goTo(afterService())} />
      </DnsStep>
      {showEnv && (
        <DnsStep title="Environment" state={stateFor("env")} onChange={() => goTo("env")}>
          <WizardEnvStep onDone={() => goTo(afterEnv)} />
        </DnsStep>
      )}
      {!isAttach && (
        <DnsStep
          title="Reach"
          description="Optional — give this service a hostname now, or do it later from the stack page."
          state={stateFor("reach")}
          summary={exposureHostname ?? "Skipped"}
          onChange={() => goTo("reach")}
        >
          <WizardReachStep onDone={() => goTo("done")} onSkip={() => goTo("done")} />
        </DnsStep>
      )}
      <DnsStep title="Done" state={stateFor("done")} last>
        {stackId && <WizardDoneStep />}
      </DnsStep>
    </ol>
  );
};

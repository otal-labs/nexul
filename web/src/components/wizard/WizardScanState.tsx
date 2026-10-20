import { LoadingDisplay } from "@/components/LoadingDisplay";
import { Button } from "@/components/ui/button";
import { RepositoryScanError } from "@/components/wizard/RepositoryScanError";
import { WizardFooter } from "@/components/wizard/WizardFooter";
import { WizardManualDockerfile } from "@/components/wizard/WizardManualDockerfile";
import { WizardSkipLink } from "@/components/wizard/WizardSkipLink";
import { useWizardGoTo } from "@/hooks/useWizardNavigation";
import { manualCandidate } from "@/models/Repository";
import { useProjectWizardStore } from "@/stores/projectWizardStore";

interface WizardScanStateProps {
  view: "scanning" | "scanFailed" | "nothingFound";
  step: "deploy" | "service";
  onBack: () => void;
  // Deploy's Later, or Service's Skip for now; absent where the door has no way past the step.
  onLeave?: (() => void) | undefined;
}

// What Deploy and Service show while the attached repository is being read, when the read failed, and when it found nothing to build.
export const WizardScanState = ({ view, step, onBack, onLeave }: WizardScanStateProps) => {
  const goTo = useWizardGoTo();
  const reading = useProjectWizardStore((s) => s.seed.repository);
  const error = useProjectWizardStore((s) => s.seed.error);
  const repository = useProjectWizardStore((s) => s.repository);
  const setCandidate = useProjectWizardStore((s) => s.setCandidate);

  return (
    <div>
      {view === "scanning" && <LoadingDisplay label={`Reading ${reading ?? "the repository"}`} />}
      {view === "scanFailed" && <RepositoryScanError error={error} />}
      {view === "nothingFound" && repository && (
        <WizardManualDockerfile
          onContinue={(path) => {
            setCandidate(manualCandidate(repository.name, path));
            goTo("service");
          }}
        />
      )}
      <WizardFooter
        onBack={onBack}
        skip={
          <>
            {view !== "scanning" && onLeave && step === "deploy" && (
              <Button type="button" variant="ghost" className="text-muted-foreground" onClick={onLeave}>
                Later
              </Button>
            )}
            {view !== "scanning" && onLeave && step === "service" && <WizardSkipLink onClick={onLeave} />}
          </>
        }
      >
        {view === "scanFailed" && (
          <Button type="button" onClick={() => goTo("repository")}>
            Go to Repository
          </Button>
        )}
      </WizardFooter>
    </div>
  );
};

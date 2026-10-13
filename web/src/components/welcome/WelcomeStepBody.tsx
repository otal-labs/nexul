import { ComputerStep } from "@/components/welcome/ComputerStep";
import { DeployServerStep } from "@/components/welcome/DeployServerStep";
import { GitHubStep } from "@/components/welcome/GitHubStep";
import { PhoneStep } from "@/components/welcome/PhoneStep";
import { ProfileStep } from "@/components/welcome/ProfileStep";
import { ProjectsStep } from "@/components/welcome/ProjectsStep";
import type { WizardStep } from "@/models/WelcomeWizard";

interface WelcomeStepBodyProps {
  step: WizardStep;
}

export const WelcomeStepBody = ({ step }: WelcomeStepBodyProps) => (
  <>
    {step === "profile" && <ProfileStep />}
    {step === "github" && <GitHubStep />}
    {step === "computer" && <ComputerStep />}
    {step === "projects" && <ProjectsStep />}
    {step === "deploy_server" && <DeployServerStep />}
    {step === "phone" && <PhoneStep />}
  </>
);

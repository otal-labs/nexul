import { AddRunnerForm } from "@/components/runner/AddRunnerForm";
import { RunnerInstallCommands } from "@/components/runner/RunnerInstallCommands";
import { AlreadyDone } from "@/components/welcome/AlreadyDone";
import { ContinueButton, WelcomeFooter } from "@/components/welcome/WelcomeFooter";
import { useCreateRunnerEnrollment } from "@/hooks/RunnerHooks";
import { useDoneOnArrival, useWelcomeFlow } from "@/hooks/useWelcomeFlow";

// Add runner's dialog in place: name it, run its command, and Continue waits for a server this person added to connect.
const AddDeployServer = () => {
  const enroll = useCreateRunnerEnrollment();
  const connected = useWelcomeFlow().marks.deploy_server === "done";
  return (
    <>
      {!enroll.data && <AddRunnerForm pending={enroll.isPending} onSubmit={(input) => enroll.mutateAsync(input)} />}
      {enroll.data && <RunnerInstallCommands enrollment={enroll.data} name={enroll.variables?.name ?? ""} />}
      <WelcomeFooter skip={!connected}>{enroll.data && <ContinueButton disabled={!connected} />}</WelcomeFooter>
    </>
  );
};

export const DeployServerStep = () => {
  const arrivedDone = useDoneOnArrival("deploy_server");
  return (
    <>
      {arrivedDone && <AlreadyDone title="A deploy server you added is connected">See it and add more on Runners.</AlreadyDone>}
      {arrivedDone && (
        <WelcomeFooter skip={false}>
          <ContinueButton />
        </WelcomeFooter>
      )}
      {!arrivedDone && <AddDeployServer />}
    </>
  );
};

import { Button } from "@/components/ui/button";
import { useCancelDeploy } from "@/hooks/DeployHooks";

interface DeployCancelButtonProps {
  deployId: string;
}

export const DeployCancelButton = ({ deployId }: DeployCancelButtonProps) => {
  const cancel = useCancelDeploy();
  return (
    <Button type="button" variant="outline" onClick={() => cancel.mutate(deployId)} loading={cancel.isPending}>
      Cancel deploy
    </Button>
  );
};

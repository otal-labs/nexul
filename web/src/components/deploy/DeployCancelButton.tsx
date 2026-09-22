import { Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useCancelDeploy } from "@/hooks/DeployHooks";

interface DeployCancelButtonProps {
  deployId: string;
}

export const DeployCancelButton = ({ deployId }: DeployCancelButtonProps) => {
  const cancel = useCancelDeploy();
  return (
    <Button type="button" variant="outline" onClick={() => cancel.mutate(deployId)} disabled={cancel.isPending}>
      {cancel.isPending && <Loader2 className="size-4 animate-spin motion-reduce:animate-none" aria-hidden />}
      Cancel deployment
    </Button>
  );
};

import { RefreshCw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { usePairNow } from "@/hooks/ComputerHooks";
import type { Computer } from "@/models/Pairing";

interface PairAgainButtonProps {
  computer: Computer;
}

export const PairAgainButton = ({ computer }: PairAgainButtonProps) => {
  const pairNow = usePairNow(computer);
  return (
    <Button type="button" variant="outline" size="sm" loading={pairNow.isPending} onClick={() => pairNow.mutate()}>
      <RefreshCw />
      Pair again
    </Button>
  );
};

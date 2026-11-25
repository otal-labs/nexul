import { Wrench } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { useListComputers } from "@/hooks/PairingHooks";
import { useWorkspacePath } from "@/hooks/useWorkspacePath";
import { computerSetupPath } from "@/models/Pairing";

interface SetupRefusalLinkProps {
  computerId: string;
}

// The fix for a setup refusal; the list holds only the viewer's own computers, so nobody else sees a way in.
export const SetupRefusalLink = ({ computerId }: SetupRefusalLinkProps) => {
  const wsPath = useWorkspacePath();
  const { data: computers } = useListComputers();
  const computer = computers?.find((c) => c.id === computerId);
  if (!computer) return null;

  return (
    <Button asChild variant="outline" size="sm" className="max-w-full">
      <Link to={wsPath(computerSetupPath(computer.id))}>
        <Wrench aria-hidden />
        <span className="truncate">Set up {computer.name}</span>
      </Link>
    </Button>
  );
};

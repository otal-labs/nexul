import { ComputersSection } from "@/components/settings/ComputersSection";
import { Button } from "@/components/ui/button";
import { useListComputers } from "@/hooks/PairingHooks";
import { stillPairing } from "@/models/Pairing";

interface SetupT3CodeStepProps {
  onFinish: () => void;
}

// Required: agent work only runs on a paired computer, so setup ends once one is paired.
export const SetupT3CodeStep = ({ onFinish }: SetupT3CodeStepProps) => {
  const { data: computers } = useListComputers();
  const paired = !!computers?.some((c) => !stillPairing(c));
  return (
    <div className="space-y-6">
      <ComputersSection bare />
      {!paired && <p className="text-sm text-muted-foreground">Pair a computer to finish setup.</p>}
      <Button className="w-full" onClick={onFinish} disabled={!paired}>
        Finish setup
      </Button>
    </div>
  );
};

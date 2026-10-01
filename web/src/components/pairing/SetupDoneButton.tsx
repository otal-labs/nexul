import { Button } from "@/components/ui/button";
import { useFetchComputerSetup, useSaveSetupChoices } from "@/hooks/ComputerSetupHooks";
import { useSetupChoices } from "@/hooks/useSetupChoices";
import type { ComputerSetup } from "@/models/Pairing";

interface SaveAndCloseProps {
  setup: ComputerSetup;
  onDone: () => void;
}

// Saves only once the harness listed its providers: an unreachable computer would otherwise save every choice away.
const SaveAndClose = ({ setup, onDone }: SaveAndCloseProps) => {
  const { choices, body } = useSetupChoices(setup);
  const save = useSaveSetupChoices(setup.computer_id);
  const done = () => {
    if (choices.length === 0) {
      onDone();
      return;
    }
    save.mutate(body, { onSuccess: onDone });
  };
  return (
    <Button type="button" onClick={done} loading={save.isPending}>
      Done
    </Button>
  );
};

interface SetupDoneButtonProps {
  computerId: string;
  onDone: () => void;
}

// The Set up step's Done: keeps the switches, models, options, and folder for this computer, then closes; it shows once setup has loaded.
export const SetupDoneButton = ({ computerId, onDone }: SetupDoneButtonProps) => {
  const { data: setup } = useFetchComputerSetup(computerId);
  return setup && <SaveAndClose setup={setup} onDone={onDone} />;
};

import { errorMessage } from "@/api/client";
import { SetupRefusalLink } from "@/components/pairing/SetupRefusalLink";
import { setupRefusalComputerId } from "@/models/Pairing";

interface PlayRunErrorProps {
  error: unknown;
}

// Why a press was refused, with the way to fix it when the refusal names a computer to set up.
export const PlayRunError = ({ error }: PlayRunErrorProps) => {
  const setupComputerId = setupRefusalComputerId(error);
  return (
    <div role="alert" className="space-y-2">
      <p className="text-sm text-destructive">{errorMessage(error)}</p>
      {setupComputerId && <SetupRefusalLink computerId={setupComputerId} />}
    </div>
  );
};

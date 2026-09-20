import { Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { ConnectorsSection } from "@/components/settings/ConnectorsSection";

interface ConnectToolsStepProps {
  onFinish: () => void;
  finishing: boolean;
}

// `bare` drops ConnectorsSection's own card chrome since WizardLayout already frames the title/subtitle.
export const ConnectToolsStep = ({ onFinish, finishing }: ConnectToolsStepProps) => (
  <div className="space-y-6">
    <ConnectorsSection bare />
    <Button className="w-full" onClick={onFinish} disabled={finishing}>
      {finishing && <Loader2 className="size-4 animate-spin motion-reduce:animate-none" aria-hidden />}
      {finishing ? "Finishing…" : "Finish setup"}
    </Button>
  </div>
);

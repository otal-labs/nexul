import { ClipboardList } from "lucide-react";

import { Button } from "@/components/ui/button";

interface WizardInterviewOfferProps {
  projectName: string;
  onStart: () => void;
  onSkip: () => void;
}

export const WizardInterviewOffer = ({ projectName, onStart, onSkip }: WizardInterviewOfferProps) => (
  <div className="animate-in fade-in-0 slide-in-from-bottom-1 space-y-3 border-t border-border pt-5 duration-200 ease-out">
    <div className="flex items-start gap-2">
      <ClipboardList className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
      <div className="space-y-1">
        <p className="text-sm font-medium">Run the interview</p>
        <p className="text-sm text-muted-foreground">
          An agent asks you one question at a time, each with a recommended answer, and records {projectName}'s
          stack, paradigm, testing strategy, principles, and vocabulary as rules every agent turn here follows. It can
          read the codebase for answers first.
        </p>
      </div>
    </div>
    <div className="flex flex-wrap gap-3">
      <Button onClick={onStart}>Start the interview</Button>
      <Button variant="ghost" onClick={onSkip}>
        Skip
      </Button>
    </div>
  </div>
);

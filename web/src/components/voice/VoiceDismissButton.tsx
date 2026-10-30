import { X } from "lucide-react";

import { Button } from "@/components/ui/button";

interface VoiceDismissButtonProps {
  onClick: () => void;
}

export const VoiceDismissButton = ({ onClick }: VoiceDismissButtonProps) => (
  <Button size="icon" variant="ghost" className="size-8" aria-label="Dismiss" title="Dismiss" onClick={onClick}>
    <X aria-hidden />
  </Button>
);

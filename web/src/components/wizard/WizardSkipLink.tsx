import { Button } from "@/components/ui/button";

export const WizardSkipLink = ({ onClick }: { onClick: () => void }) => (
  <Button type="button" variant="ghost" className="text-muted-foreground" onClick={onClick}>
    Skip for now
  </Button>
);

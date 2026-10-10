import { desktopSteps } from "@/utils/PairCommands";

interface DesktopPairStepsProps {
  viaTunnel: boolean;
}

// The desktop app needs no terminal: its Settings make the pairing link.
export const DesktopPairSteps = ({ viaTunnel }: DesktopPairStepsProps) => (
  <ol className="list-decimal space-y-1.5 pl-5 text-sm text-muted-foreground marker:text-muted-foreground/70">
    {desktopSteps(viaTunnel).map((step) => (
      <li key={step}>{step}</li>
    ))}
  </ol>
);

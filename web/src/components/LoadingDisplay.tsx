import { Loader2 } from "lucide-react";

interface LoadingDisplayProps {
  label?: string;
}

export const LoadingDisplay = ({ label = "Loading" }: LoadingDisplayProps) => (
  <div
    role="status"
    className="animate-in fade-in-0 flex items-center justify-center gap-2 p-8 text-muted-foreground duration-150 ease-out"
  >
    <Loader2 className="size-4 animate-spin motion-reduce:animate-none" aria-hidden />
    <span className="text-sm">{label}</span>
  </div>
);

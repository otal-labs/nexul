import { EmptyRow } from "@/components/EmptyRow";
import { SettingsCard } from "@/components/settings/SettingsCard";

interface StackEnvironmentKeysProps {
  keys: string[];
  description: string;
}

// The variables' names with their values withheld: for someone who cannot change the stack, and for a branch deployment.
export const StackEnvironmentKeys = ({ keys, description }: StackEnvironmentKeysProps) => (
  <SettingsCard id="environment" title="Environment" description={description}>
    {keys.length === 0 && <EmptyRow flush>No variables are set.</EmptyRow>}
    {keys.length > 0 && (
      <ul className="divide-y divide-border rounded-md border border-border">
        {keys.map((key) => (
          <li key={key} className="flex items-center justify-between gap-3 px-3 py-2">
            <span className="font-mono text-sm wrap-anywhere">{key}</span>
            <span className="shrink-0 font-mono text-xs text-muted-foreground">
              <span aria-hidden>••••••••</span>
              <span className="sr-only">value hidden</span>
            </span>
          </li>
        ))}
      </ul>
    )}
  </SettingsCard>
);

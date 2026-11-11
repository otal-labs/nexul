import { CommandBlock } from "@/components/pairing/CommandBlock";
import { useFetchSettings } from "@/hooks/AuthHooks";
import type { AutomationVersionDiff } from "@/models/AutomationVersion";

interface AutomationSdkSetupProps {
  automationId: string;
  /** The automation's pending and active versions; absent right after it is created. */
  versions?: AutomationVersionDiff | undefined;
}

// pull starts from the pending version's source so a push builds on it, else the active one's; init asks for a token.
export const AutomationSdkSetup = ({ automationId, versions }: AutomationSdkSetupProps) => {
  const { data: settings } = useFetchSettings();
  const instanceUrl = settings?.instance_url || window.location.origin;
  const existing = !!(versions?.pending || versions?.active);
  const from = [versions?.pending, versions?.active].find((version) => version?.source);
  const lead = existing && !from ? "No version kept its source, so put your own copy in " : "Edit ";
  const where = versions ? "here" : "on Versions";
  const tail = existing ? ", then push and merge it here." : `, push, then merge it ${where} and switch it on.`;
  const getLines = [
    "bun add @otal-labs/nexul-sdk",
    `bunx nexul init --url ${instanceUrl} --id ${automationId}`,
    ...(from ? [`bunx nexul pull ${automationId}${from === versions?.pending ? ` ${from.id}` : ""}`] : []),
  ];

  return (
    <div className="space-y-3">
      <CommandBlock shell="Get the code" lines={getLines} />
      <p className="text-sm text-muted-foreground">
        {lead}
        <code className="font-mono text-xs">src/index.ts</code>
        {tail}
      </p>
      <CommandBlock shell="Push" lines={[`bunx nexul push ${automationId} "${existing ? "<what changed>" : "First version"}"`]} />
    </div>
  );
};

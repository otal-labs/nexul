import { useFetchSettings } from "@/hooks/AuthHooks";

interface AutomationSdkSetupProps {
  automationId: string;
}

// The commands that take an empty folder to this automation's first pending version; init asks for a personal access token.
export const AutomationSdkSetup = ({ automationId }: AutomationSdkSetupProps) => {
  const { data: settings } = useFetchSettings();
  const instanceUrl = settings?.instance_url || "https://your-instance.example.com";

  return (
    <div className="terminal-window">
      <div className="terminal-window__bar">
        <span className="terminal-window__dot" aria-hidden />
        <span className="terminal-window__dot" aria-hidden />
        <span className="terminal-window__dot" aria-hidden />
        <span className="terminal-window__title">Write its code, in an empty folder</span>
      </div>
      <pre className="terminal-window__body whitespace-pre-wrap break-all">
        bun add @otal-labs/nexul-sdk
        {"\n"}bunx nexul init --url {instanceUrl} --id {automationId}
        {"\n"}bunx nexul push {automationId} &quot;First version&quot;
      </pre>
    </div>
  );
};

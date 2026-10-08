import { Badge } from "@/components/ui/badge";
import { EmptyRow } from "@/components/EmptyRow";
import { SettingsCard } from "@/components/settings/SettingsCard";

interface AutomationSubscriptionsSectionProps {
  subscriptions: string[];
}

// Subscriptions are declared in code and announced at dial-in — the UI displays them, never edits them.
export const AutomationSubscriptionsSection = ({ subscriptions }: AutomationSubscriptionsSectionProps) => (
  <SettingsCard id="subscriptions" title="Subscriptions" description="The events this automation reacts to, declared in its code.">
    {subscriptions.length === 0 && <EmptyRow>This automation isn't subscribed to anything</EmptyRow>}
    {subscriptions.length > 0 && (
      <div className="flex flex-wrap gap-1.5">
        {subscriptions.map((topic) => (
          <Badge key={topic} variant="outline" className="font-mono">
            {topic}
          </Badge>
        ))}
      </div>
    )}
  </SettingsCard>
);

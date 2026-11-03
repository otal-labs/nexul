import { ComputerFact } from "@/components/settings/ComputerFact";
import { SettingsStatus } from "@/components/settings/SettingsStatus";
import { Button } from "@/components/ui/button";
import { useSetGitIdentity } from "@/hooks/ComputerToolHooks";
import type { ComputerFacts } from "@/models/ComputerFacts";
import { gitAuthor } from "@/models/ComputerTools";
import type { Computer } from "@/models/Pairing";

// What git on the computer lacks of an identity, "" when it has both or Nexul can't tell: a runner older than tool
// reports never says whether git is there.
const identityGap = (facts: ComputerFacts) => {
  const git = facts.tools?.find((t) => t.name === "git");
  if (!git?.path || (facts.git_name && facts.git_email)) return "";
  if (!facts.git_name && !facts.git_email) return "Not set";
  if (!facts.git_name) return "Name not set";
  return "Email not set";
};

interface ComputerGitIdentityFactProps {
  computer: Computer;
  facts: ComputerFacts;
}

// Git's name and email on the computer, and while either is missing, the offer to take them from the GitHub sign-in.
export const ComputerGitIdentityFact = ({ computer, facts }: ComputerGitIdentityFactProps) => {
  const { setIdentity, isPending } = useSetGitIdentity(computer);
  const author = gitAuthor({ name: facts.git_name, email: facts.git_email });
  const gap = identityGap(facts);
  if (!author && !gap) return null;
  return (
    <ComputerFact label="Git identity">
      <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
        {author && <span className="break-all">{author}</span>}
        {gap && <SettingsStatus tone="warning">{gap}</SettingsStatus>}
        {gap && (
          <Button type="button" variant="outline" size="sm" className="h-7" loading={isPending} onClick={() => void setIdentity()}>
            Use my GitHub identity
          </Button>
        )}
      </span>
    </ComputerFact>
  );
};

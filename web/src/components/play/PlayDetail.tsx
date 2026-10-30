import { AutoPlaysPanel } from "@/components/autoplay/AutoPlaysPanel";
import { Container } from "@/components/Container";
import { PlayDetailHeader } from "@/components/play/PlayDetailHeader";
import { PlayFactsStrip } from "@/components/play/PlayFactsStrip";
import { PlayInstructionsSection } from "@/components/play/PlayInstructionsSection";
import { PlayRunsSection } from "@/components/play/PlayRunsSection";
import { useAreaAccess } from "@/hooks/AccessHooks";
import type { WorkspacePlay } from "@/models/Play";

interface PlayDetailProps {
  play: WorkspacePlay;
}

// One page in the order a lead reads a play: what it is, when it starts by itself, how its runs went, what it says.
export const PlayDetail = ({ play }: PlayDetailProps) => {
  const can = useAreaAccess();
  // An interview play takes no auto plays.
  const autoPlays = play.type !== "interview" && !!can?.("autoPlays");

  return (
    <Container className="space-y-8 py-8 sm:pr-3.5">
      <PlayDetailHeader play={play} />
      <PlayFactsStrip play={play} />
      {autoPlays && <AutoPlaysPanel play={play} />}
      <PlayRunsSection play={play} />
      <PlayInstructionsSection play={play} />
    </Container>
  );
};

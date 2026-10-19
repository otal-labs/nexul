import { AutoPlaysPanel } from "@/components/autoplay/AutoPlaysPanel";
import { Container } from "@/components/Container";
import { PageTabs, PageTabsContent } from "@/components/PageTabs";
import { PlayDetailHeader } from "@/components/play/PlayDetailHeader";
import { PlayEnabledSwitch } from "@/components/play/PlayEnabledSwitch";
import { PlayFacts } from "@/components/play/PlayFacts";
import { useAreaAccess } from "@/hooks/AccessHooks";
import type { Play } from "@/models/Play";

interface PlayDetailProps {
  play: Play;
}

export const PlayDetail = ({ play }: PlayDetailProps) => {
  const can = useAreaAccess();
  // An interview play takes no auto plays.
  const noAutoPlays = play.type === "interview" || !can?.("autoPlays");

  return (
    <Container className="space-y-6 py-8">
      <PlayDetailHeader play={play} />
      <PageTabs
        label="Play sections"
        trailing={<PlayEnabledSwitch play={play} />}
        tabs={[
          { value: "play", label: "Play" },
          { value: "auto-plays", label: "Auto plays", hidden: noAutoPlays },
        ]}
      >
        <PageTabsContent value="play">
          <PlayFacts play={play} />
        </PageTabsContent>
        <PageTabsContent value="auto-plays">
          <AutoPlaysPanel play={play} />
        </PageTabsContent>
      </PageTabs>
    </Container>
  );
};

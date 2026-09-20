import { Container } from "@/components/Container";
import { PageHeader } from "@/components/PageHeader";
import { TopologyCanvas } from "@/components/topology/TopologyCanvas";

export const TopologyPage = () => (
  <Container className="py-6">
    <PageHeader
      title="Topology"
      subtitle="Every service, the network it runs on, and how traffic reaches it."
      className="mb-4"
    />
    {/* Reserved chrome height grows on narrow viewports since the subtitle wraps to 2-3 lines below `sm`. */}
    <div className="h-[calc(100vh-11.5rem)] overflow-hidden rounded-xl border border-border bg-card shadow-card sm:h-[calc(100vh-10rem)] md:h-[calc(100vh-9.5rem)]">
      <TopologyCanvas />
    </div>
  </Container>
);

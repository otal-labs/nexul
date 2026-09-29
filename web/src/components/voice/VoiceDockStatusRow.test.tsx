import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { VoiceDockStatusRow } from "@/components/voice/VoiceDockStatusRow";

const access = vi.hoisted(() => ({ sections: [] as string[] }));
vi.mock("@/hooks/AccessHooks", () => ({ useCanOpenSection: (section: string) => access.sections.includes(section) }));

const renderRow = () =>
  render(
    <MemoryRouter>
      <VoiceDockStatusRow status="not_configured" channelName="general" onLeave={() => {}} />
    </MemoryRouter>,
  );

beforeEach(() => {
  access.sections = [];
});

describe("VoiceDockStatusRow", () => {
  it("links the missing LiveKit setup to Connectors for a viewer who can open it", () => {
    access.sections = ["connectors"];
    renderRow();
    expect(screen.getByRole("link", { name: "LiveKit setup needed" })).toHaveAttribute(
      "href",
      "/configuration/connectors",
    );
  });

  it("names the missing LiveKit setup without a link for a viewer who can't open Connectors", () => {
    renderRow();
    expect(screen.getByText("LiveKit setup needed")).toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });
});

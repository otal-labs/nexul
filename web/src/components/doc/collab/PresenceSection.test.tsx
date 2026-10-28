import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render as rtlRender, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { avatarFill } from "@nexul/client-core/avatarFill";

import { api } from "@/api/client";
import { PresenceSection } from "@/components/doc/collab/PresenceSection";
import type { CollabParticipant } from "@/components/doc/collab/useCollabSession";
import { useWorkspaceStore } from "@/stores/workspaceStore";

vi.mock("@/api/client", () => ({ api: { get: vi.fn() } }));

const render = (ui: ReactNode) => rtlRender(<QueryClientProvider client={new QueryClient()}>{ui}</QueryClientProvider>);

beforeEach(() => {
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  vi.mocked(api.get).mockResolvedValue({ data: { people: [{ user_id: "u-alice", login: "alice", name: "Alice", avatar_url: "" }] } });
});

const participants: CollabParticipant[] = [
  { clientID: 1, name: "Alice", color: "#ef4444", activity: "editing" },
  { clientID: 2, name: "Bob", color: "#22c55e", activity: "viewing" },
];

describe("PresenceSection", () => {
  it("renders nothing while nobody is present", () => {
    const { container } = render(<PresenceSection participants={[]} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("shows an avatar per participant with their activity, the image when one carries it and initials otherwise", () => {
    const withAvatar: CollabParticipant[] = [
      { clientID: 1, name: "Alice", color: "#ef4444", activity: "editing", avatar: "https://avatars.example/alice.png" },
      { clientID: 2, name: "Bob", color: "#22c55e", activity: "viewing" },
    ];
    render(<PresenceSection participants={withAvatar} />);
    const bubbles = screen.getAllByTestId("presence-avatar");
    expect(bubbles).toHaveLength(2);
    expect(screen.getByLabelText("Alice editing")).toBeInTheDocument();
    expect(screen.getByLabelText("Bob viewing")).toBeInTheDocument();
    expect(bubbles[0]?.querySelector("img")).toHaveAttribute("src", "https://avatars.example/alice.png");
    expect(bubbles[1]?.querySelector("img")).toBeNull();
    expect(bubbles[1]).toHaveTextContent("B");
  });

  it("fills a known account's avatar as the person's avatar is filled everywhere, not with the session's cursor colour", async () => {
    render(<PresenceSection participants={[{ clientID: 7, id: "u-alice", name: "Alice", color: "#ef4444", activity: "editing" }, participants[1]!]} />);
    const [alice, bob] = screen.getAllByTestId("presence-avatar");
    await vi.waitFor(() => expect(alice).toHaveStyle({ backgroundColor: avatarFill("alice") }));
    expect(bob).toHaveStyle({ backgroundColor: "#22c55e" });
  });

  it("caps the stack and shows an overflow chip beyond max", () => {
    const many: CollabParticipant[] = [
      ...participants,
      { clientID: 3, name: "Cara", color: "#0ea5e9", activity: "editing" },
      { clientID: 4, name: "Dan", color: "#8b5cf6", activity: "viewing" },
      { clientID: 5, name: "Eve", color: "#6366f1", activity: "editing" },
    ];
    render(<PresenceSection participants={many} max={3} />);
    expect(screen.getAllByTestId("presence-avatar")).toHaveLength(3);
    expect(screen.getByLabelText("2 more")).toBeInTheDocument();
  });
});

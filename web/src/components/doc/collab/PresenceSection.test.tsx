import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PresenceSection } from "@/components/doc/collab/PresenceSection";
import type { CollabParticipant } from "@/components/doc/collab/useCollabSession";

const participants: CollabParticipant[] = [
  { clientID: 1, name: "Alice", color: "#ef4444", activity: "editing" },
  { clientID: 2, name: "Bob", color: "#22c55e", activity: "viewing" },
];

describe("PresenceSection", () => {
  it("renders nothing while nobody is present", () => {
    const { container } = render(<PresenceSection participants={[]} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders the avatar image when a participant carries one, initials otherwise", () => {
    const withAvatar: CollabParticipant[] = [
      { clientID: 1, name: "Alice", color: "#ef4444", activity: "editing", avatar: "https://avatars.example/alice.png" },
      { clientID: 2, name: "Bob", color: "#22c55e", activity: "viewing" },
    ];
    render(<PresenceSection participants={withAvatar} />);
    const bubbles = screen.getAllByTestId("presence-avatar");
    expect(bubbles[0]?.querySelector("img")).toHaveAttribute("src", "https://avatars.example/alice.png");
    expect(bubbles[1]?.querySelector("img")).toBeNull();
    expect(bubbles[1]).toHaveTextContent("B");
  });

  it("shows an avatar per participant with their activity", () => {
    render(<PresenceSection participants={participants} />);
    expect(screen.getAllByTestId("presence-avatar")).toHaveLength(2);
    expect(screen.getByLabelText("Alice editing")).toBeInTheDocument();
    expect(screen.getByLabelText("Bob viewing")).toBeInTheDocument();
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

import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { VoiceRoomAvatars } from "@/components/voice/VoiceRoomAvatars";

vi.mock("@/hooks/PeopleHooks", () => ({
  usePersonLookup: () => (identity: string) => ({ user_id: identity, login: identity, display_name: "", avatar_url: "" }),
}));

const occupants = [
  { identity: "u2", name: "Lena Kowalski" },
  { identity: "u3", name: "Sam Okafor" },
];

describe("VoiceRoomAvatars", () => {
  it("rings only the people who are speaking", () => {
    render(<VoiceRoomAvatars occupants={occupants} speaking={new Set(["u3"])} />);

    expect(screen.getByTitle("Sam Okafor")).toHaveClass("ring-success");
    expect(screen.getByTitle("Lena Kowalski")).not.toHaveClass("ring-success");
  });

  it("rings nobody when no one is speaking", () => {
    render(<VoiceRoomAvatars occupants={occupants} />);

    expect(screen.getByTitle("Sam Okafor")).not.toHaveClass("ring-success");
  });
});

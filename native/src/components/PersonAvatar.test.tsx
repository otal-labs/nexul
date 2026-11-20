import { render, screen } from "@testing-library/react-native";
import { PixelRatio } from "react-native";

import { PersonAvatar } from "@/components/PersonAvatar";
import { useSessionStore } from "@/stores/sessionStore";

jest.mock("expo-sqlite/kv-store", () => ({
  __esModule: true,
  default: { getItemSync: () => null, setItemSync: jest.fn(), removeItemSync: jest.fn() },
}));

// An uploaded picture may be a 10MB animated GIF, which froze the board drawing one per card on the UI thread.
test("an uploaded avatar is asked for as a still at the pixels its box takes, never as the upload itself", async () => {
  useSessionStore.setState({ host: "https://nexul.example.com" });
  await render(
    <PersonAvatar person={{ user_id: "u1", login: "alice", display_name: "Alice", avatar_url: "/api/people/u1/avatar?v=abc" }} size={22} />,
  );
  const [source] = (screen.toJSON() as { props: Record<string, unknown> }).props.source as [{ uri: string }];
  expect(source.uri).toBe(`https://nexul.example.com/api/people/u1/avatar?v=abc&size=${Math.ceil(22 * PixelRatio.get())}`);
});

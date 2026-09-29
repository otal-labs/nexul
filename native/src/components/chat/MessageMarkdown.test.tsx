import { render, screen, userEvent } from "@testing-library/react-native";
import { Linking } from "react-native";

import { MessageMarkdown } from "@/components/chat/MessageMarkdown";

const mockPush = jest.fn();
jest.mock("expo-router", () => ({ router: { push: (href: string) => mockPush(href) } }));
jest.mock("@/stores/sessionStore", () => ({
  useSessionStore: { getState: () => ({ host: "https://nexul.example.com" }) },
}));

// Stands in for the native view, which reports a tapped link as { url }; here the whole markdown is that one link.
jest.mock("react-native-enriched-markdown", () => {
  const { Text } = jest.requireActual("react-native");
  return {
    EnrichedMarkdownText: ({ markdown, onLinkPress }: { markdown: string; onLinkPress: (e: { url: string }) => void }) => (
      <Text role="link" onPress={() => onLinkPress({ url: markdown })}>
        {markdown}
      </Text>
    ),
  };
});

const openURL = jest.spyOn(Linking, "openURL").mockResolvedValue(true);

beforeEach(() => {
  mockPush.mockReset();
  openURL.mockClear();
});

test.each([
  ["a doc by id", "/docs/doc-2", "/more/docs/doc-2"],
  ["a doc under its project token", "/docs/CHK/doc-2", "/more/docs/doc-2"],
  ["a ticket", "/tickets/t-9", "/board/ticket/t-9"],
  ["a doc on the instance", "https://nexul.example.com/docs/CHK/doc-2", "/more/docs/doc-2"],
  ["a ticket on the instance with a fragment", "https://nexul.example.com/tickets/t-9#thread", "/board/ticket/t-9"],
])("%s opens in the app", async (_, url, route) => {
  await render(<MessageMarkdown markdown={url} />);

  await userEvent.press(screen.getByRole("link"));

  expect(mockPush).toHaveBeenCalledWith(route);
  expect(openURL).not.toHaveBeenCalled();
});

test.each([
  ["an external site", "https://example.com/status", "https://example.com/status"],
  ["another web page of the instance", "/board/CHK", "https://nexul.example.com/board/CHK"],
  ["a deeper docs path", "/docs/CHK/doc-2/history", "https://nexul.example.com/docs/CHK/doc-2/history"],
  ["a doc on another instance", "https://other.example.com/docs/doc-2", "https://other.example.com/docs/doc-2"],
])("%s opens in the browser", async (_, url, opened) => {
  await render(<MessageMarkdown markdown={url} />);

  await userEvent.press(screen.getByRole("link"));

  expect(openURL).toHaveBeenCalledWith(opened);
  expect(mockPush).not.toHaveBeenCalled();
});

import { render, screen, userEvent } from "@testing-library/react-native";
import { Linking } from "react-native";

import { MessageMarkdown } from "@/components/chat/MessageMarkdown";

const mockPush = jest.fn();
jest.mock("expo-router", () => ({ router: { push: (href: unknown) => mockPush(href) } }));
jest.mock("@/stores/sessionStore", () => ({
  useSessionStore: { getState: () => ({ host: "https://nexul.example.com" }) },
}));

let mockCanReadTickets = true;
jest.mock("@/hooks/WorkspaceHooks", () => ({ useAreaAccess: () => (area: string) => area === "tickets" && mockCanReadTickets }));

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
  mockCanReadTickets = true;
  mockPush.mockReset();
  openURL.mockClear();
});

const keyRoute = (id: string, workspace: string) => ({ pathname: "/board/ticket/[id]", params: { id, workspace } });

test.each<[string, string, unknown]>([
  ["a mentioned doc", "/docs/doc-2", "/more/docs/doc-2"],
  ["a mentioned ticket", "/tickets/t-9", "/board/ticket/t-9"],
  ["a doc page", "/otal/docs/CHK/doc-2", "/more/docs/doc-2"],
  ["a doc page on the instance", "https://nexul.example.com/otal/docs/CHK/doc-2", "/more/docs/doc-2"],
  ["a ticket page, whose key needs its workspace", "/otal/tickets/WEB-1", keyRoute("WEB-1", "otal")],
  ["a ticket page on the instance with a fragment", "https://nexul.example.com/otal/tickets/WEB-1#thread", keyRoute("WEB-1", "otal")],
])("%s opens in the app", async (_, url, route) => {
  await render(<MessageMarkdown markdown={url} />);

  await userEvent.press(screen.getByRole("link"));

  expect(mockPush).toHaveBeenCalledWith(route);
  expect(openURL).not.toHaveBeenCalled();
});

test.each([
  ["an external site", "https://example.com/status", "https://example.com/status"],
  ["another web page of the instance", "/otal/board/CHK", "https://nexul.example.com/otal/board/CHK"],
  ["a deeper docs path", "/otal/docs/CHK/doc-2/history", "https://nexul.example.com/otal/docs/CHK/doc-2/history"],
  ["a doc on another instance", "https://other.example.com/docs/doc-2", "https://other.example.com/docs/doc-2"],
])("%s opens in the browser", async (_, url, opened) => {
  await render(<MessageMarkdown markdown={url} />);

  await userEvent.press(screen.getByRole("link"));

  expect(openURL).toHaveBeenCalledWith(opened);
  expect(mockPush).not.toHaveBeenCalled();
});

test("a ticket link opens in the browser, not the Board tab, without tickets:read", async () => {
  mockCanReadTickets = false;
  await render(<MessageMarkdown markdown="/tickets/t-9" />);

  await userEvent.press(screen.getByRole("link"));

  expect(openURL).toHaveBeenCalledWith("https://nexul.example.com/tickets/t-9");
  expect(mockPush).not.toHaveBeenCalled();
});

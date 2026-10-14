import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, userEvent } from "@testing-library/react-native";

import type { MentionChipData } from "@nexul/client-core/mention";

import { api } from "@/api/client";
import { DocBody } from "@/components/docs/DocBody";

jest.mock("@/api/client", () => ({ api: { get: jest.fn(), post: jest.fn() } }));
jest.mock("@/hooks/PeopleHooks", () => ({ useMentionName: () => () => undefined }));
jest.mock("@/hooks/WorkspaceHooks", () => ({
  useSelectedWorkspace: () => ({ id: "ws-1", name: "Acme", slug: "acme", mention_chip_template: "{ticket.Ticket} {ticket.Status}" }),
  useAreaAccess: () => () => true,
}));
jest.mock("@/stores/sessionStore", () => ({ useSessionStore: { getState: () => ({ host: "https://nexul.example.com" }) } }));

const mockPush = jest.fn();
jest.mock("expo-router", () => ({ router: { push: (href: unknown) => mockPush(href) } }));

// Stands in for the native view: a link is a pressable run that reports { url }, an inline image is its alt text's image.
jest.mock("react-native-enriched-markdown", () => {
  const { Text, View } = jest.requireActual("react-native");
  return {
    EnrichedMarkdownText: ({ markdown, onLinkPress }: { markdown: string; onLinkPress: (e: { url: string }) => void }) => (
      <View>
        {markdown.split(/(!?\[[^\]]*\]\([^)]*\))/).map((part, i) => {
          const image = /^!\[([^\]]*)\]/.exec(part);
          const link = /^\[([^\]]*)\]\(([^)]*)\)$/.exec(part);
          if (image) return <View key={i} accessible role="img" aria-label={image[1]} />;
          if (link) return <Text key={i} role="link" onPress={() => onLinkPress({ url: link[2] ?? "" })}>{link[1]}</Text>;
          return <Text key={i}>{part}</Text>;
        })}
      </View>
    ),
  };
});

const body = JSON.stringify({
  type: "doc",
  content: [
    {
      type: "paragraph",
      content: [
        { type: "text", text: "See " },
        { type: "mention", attrs: { type: "doc", id: "d-2", label: "old doc" } },
        { type: "text", text: " and " },
        { type: "mention", attrs: { type: "ticket", id: "t-9", label: "old ticket" } },
      ],
    },
  ],
});

const renderBody = async (chips: MentionChipData[]) => {
  jest.mocked(api.post).mockResolvedValue({ chips });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  await render(
    <QueryClientProvider client={client}>
      <DocBody body={body} />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  mockPush.mockReset();
  jest.mocked(api.post).mockReset();
});

test("a doc and a ticket the viewer can't open show their text, locked, and don't open", async () => {
  await renderBody([
    { type: "doc", id: "d-2", title: "Secret vault", can_open: false },
    { type: "ticket", id: "t-9", title: "Rotate keys", status_label: "In progress", can_open: false },
  ]);

  await screen.findByText(/Secret vault/);
  expect(screen.getByText(/Rotate keys In progress/)).toBeTruthy();
  expect(screen.getAllByRole("img", { name: "You don't have access" })).toHaveLength(2);
  expect(screen.queryByRole("link")).toBeNull();
  await userEvent.press(screen.getByText(/Secret vault/));
  await userEvent.press(screen.getByText(/Rotate keys/));
  expect(mockPush).not.toHaveBeenCalled();
});

test("a doc and a ticket the viewer can open stay links into the app", async () => {
  await renderBody([
    { type: "doc", id: "d-2", title: "Architecture", can_open: true },
    { type: "ticket", id: "t-9", title: "Rotate keys", status_label: "In progress", can_open: true },
  ]);

  await userEvent.press(await screen.findByRole("link", { name: "Architecture" }));
  await userEvent.press(screen.getByRole("link", { name: "Rotate keys In progress" }));

  expect(mockPush).toHaveBeenNthCalledWith(1, "/more/docs/d-2");
  expect(mockPush).toHaveBeenNthCalledWith(2, "/board/ticket/t-9");
  expect(screen.queryByRole("img", { name: "You don't have access" })).toBeNull();
});

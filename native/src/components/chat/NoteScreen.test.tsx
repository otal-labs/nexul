import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react-native";

import { api } from "@/api/client";
import { ApiError } from "@/api/errors";
import { NoteScreen } from "@/components/chat/NoteScreen";

jest.mock("expo-router", () => ({
  useLocalSearchParams: () => ({ id: "f1", name: "findings.md" }),
  Stack: { Screen: () => null },
  router: { push: jest.fn() },
}));
jest.mock("@/stores/sessionStore", () => ({
  useSessionStore: Object.assign(() => "https://nexul.example.com", { getState: () => ({ host: "https://nexul.example.com" }) }),
  readSessionToken: () => "ses_abc",
}));
jest.mock("@/hooks/WorkspaceHooks", () => ({ useAreaAccess: () => () => true }));
jest.mock("react-native-enriched-markdown", () => jest.requireActual("react-native-enriched-markdown/jest"));
jest.mock("@/api/client", () => ({ api: { get: jest.fn() } }));

const get = jest.mocked(api.get);

const renderNote = async () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  await render(
    <QueryClientProvider client={client}>
      <NoteScreen />
    </QueryClientProvider>,
  );
};

beforeEach(() => get.mockReset());

test("the note's markdown is read from its file and rendered", async () => {
  get.mockResolvedValue("## Findings\n\nThe cause is the retry loop.");
  await renderNote();

  expect(await screen.findByText(/The cause is the retry loop\./)).toBeTruthy();
  expect(get).toHaveBeenCalledWith("/api/attachments/f1");
});

test("a note deleted since the thread loaded says so", async () => {
  get.mockRejectedValue(new ApiError(404, { message: "not found" }, "GET failed: 404"));
  await renderNote();

  expect(await screen.findByText("This note doesn't exist or was deleted.")).toBeTruthy();
});

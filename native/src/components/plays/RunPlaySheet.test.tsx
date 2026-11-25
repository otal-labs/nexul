import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, userEvent, waitFor } from "@testing-library/react-native";

import type { MyWorkspaceInfo } from "@nexul/client-core/permissions";

import { api } from "@/api/client";
import { ApiError } from "@/api/errors";
import { RunPlaySheet } from "@/components/plays/RunPlaySheet";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("@/api/client", () => ({ api: { get: jest.fn(), post: jest.fn() } }));

const mockRouter = { back: jest.fn(), push: jest.fn() };
jest.mock("expo-router", () => ({
  useRouter: () => mockRouter,
  useLocalSearchParams: () => ({ playId: "pl-1", label: "Fix with AI", targetType: "ticket", targetId: "t-1", projectId: "p-1" }),
}));

let me: MyWorkspaceInfo;

const renderSheet = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <RunPlaySheet />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  me = { role_name: "Developer", permissions: ["plays:run"], show_locked_actions: false };
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  mockRouter.back.mockReset();
  mockRouter.push.mockReset();
  jest.mocked(api.post).mockReset();
  jest.mocked(api.get).mockReset().mockImplementation(async (url: string) => {
    if (url === "/api/workspaces") return [{ id: "ws-1", name: "Acme" }];
    if (url === "/api/workspaces/ws-1/me") return me;
    throw new Error(`unexpected GET ${url}`);
  });
});

describe("RunPlaySheet", () => {
  test("a run no computer of the person's can take shows why and that the web picks where, and stays open", async () => {
    const refusal = { message: "None of your computers has this project.", code: "invalid", details: { reason: "needs_location" } };
    jest.mocked(api.post).mockRejectedValue(new ApiError(400, refusal, "POST failed: 400"));
    await renderSheet();

    await userEvent.setup().press(await screen.findByRole("button", { name: "Run" }));

    expect(await screen.findByText("None of your computers has this project.")).toBeTruthy();
    expect(screen.getByText(/Pick the computer on the web once/)).toBeTruthy();
    expect(mockRouter.back).not.toHaveBeenCalled();
  });

  test("starts the play on the ticket with this run's instructions, then opens the run", async () => {
    jest.mocked(api.post).mockResolvedValue({ id: "r-9", target_type: "ticket", target_id: "t-1" });
    await renderSheet();
    const user = userEvent.setup();

    await user.type(await screen.findByLabelText("Instructions for this run"), "  Docs only.  ");
    await user.press(screen.getByRole("button", { name: "Run" }));

    await waitFor(() => expect(mockRouter.push).toHaveBeenCalledWith("/board/run/r-9"));
    expect(api.post).toHaveBeenCalledWith("/api/plays/pl-1/run", {
      target_type: "ticket",
      target_id: "t-1",
      custom_instructions: "Docs only.",
      memory_ids: [],
      model_options: [],
    });
    expect(mockRouter.back).toHaveBeenCalled();
  });

  test("a viewer whose role can't run plays gets no Run", async () => {
    me = { role_name: "Client", permissions: [], show_locked_actions: false };
    await renderSheet();

    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me"));
    expect(screen.queryByRole("button", { name: "Run" })).toBeNull();
  });
});

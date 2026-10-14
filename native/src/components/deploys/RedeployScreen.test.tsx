import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, userEvent, waitFor } from "@testing-library/react-native";

import type { MyWorkspaceInfo } from "@nexul/client-core/permissions";

import { api } from "@/api/client";
import { ApiError } from "@/api/errors";
import { RedeployScreen } from "@/components/deploys/RedeployScreen";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("@/api/client", () => ({ api: { get: jest.fn(), post: jest.fn() } }));

const mockBack = jest.fn();
jest.mock("expo-router", () => ({
  useRouter: () => ({ back: mockBack }),
  useLocalSearchParams: () => ({ stackId: "s-1", image: "api:2" }),
}));

let me: MyWorkspaceInfo;

const renderScreen = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <RedeployScreen />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  me = { role_name: "Developer", permissions: ["deploys:write"], show_locked_actions: false };
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  jest.mocked(api.get).mockReset().mockImplementation(async (url: string) => {
    if (url === "/api/workspaces") return [{ id: "ws-1", name: "Acme" }];
    if (url === "/api/workspaces/ws-1/me") return me;
    return { id: "s-1", project_id: "p-1", name: "checkout-api" };
  });
  jest.mocked(api.post).mockReset();
  mockBack.mockReset();
});

describe("RedeployScreen", () => {
  test("shows the image this confirm redeploys", async () => {
    await renderScreen();
    expect(screen.getByText("api:2")).toBeTruthy();
  });

  test("Redeploy posts the deploy and dismisses the sheet", async () => {
    jest.mocked(api.post).mockResolvedValue({ id: "d-3", stack_id: "s-1", image: "api:2", status: "pending" });
    await renderScreen();

    await userEvent.setup().press(await screen.findByRole("button", { name: "Redeploy" }));

    expect(api.post).toHaveBeenCalledWith("/api/deploys", { stack_id: "s-1", image: "api:2" });
    expect(await screen.findByText("Redeploy")).toBeTruthy();
    expect(mockBack).toHaveBeenCalled();
  });

  test("a deploy already running reads as a sentence, not the server's raw conflict", async () => {
    jest.mocked(api.post).mockRejectedValue(
      new ApiError(409, { message: 'conflict: stack "api" already has an active deploy', code: "CONFLICT" }, "POST failed: 409"),
    );
    await renderScreen();

    await userEvent.setup().press(await screen.findByRole("button", { name: "Redeploy" }));

    expect(await screen.findByText("A deploy is already running on this stack. Redeploy once it finishes.")).toBeTruthy();
    expect(screen.queryByText(/conflict:/)).toBeNull();
    expect(mockBack).not.toHaveBeenCalled();
  });

  test("Cancel dismisses the sheet without deploying", async () => {
    await renderScreen();

    await userEvent.setup().press(screen.getByRole("button", { name: "Cancel" }));

    expect(api.post).not.toHaveBeenCalled();
    expect(mockBack).toHaveBeenCalled();
  });

  test.each([
    ["hides Redeploy from a role that can't deploy", { permissions: ["stacks:read"], show_locked_actions: false }, undefined],
    ["locks Redeploy for a role that shows locked actions", { permissions: ["stacks:read"], show_locked_actions: true }, "Your role can't deploy"],
  ])("a deep link %s and sends nothing", async (_name, role, hint) => {
    me = { role_name: "Client", ...role };
    await renderScreen();

    await screen.findByText("api:2");
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me"));
    await act(async () => {});
    const control = screen.queryByRole("button", { name: "Redeploy" });
    expect(control === null).toBe(hint === undefined);
    expect(control?.props.accessibilityHint).toBe(hint);
    expect(api.post).not.toHaveBeenCalled();
  });
});

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, userEvent, waitFor } from "@testing-library/react-native";

import type { MyWorkspaceInfo } from "@nexul/client-core/permissions";

import { api } from "@/api/client";
import { ApiError } from "@/api/errors";
import { CancelDeploySheet } from "@/components/deploys/CancelDeploySheet";
import { useWorkspaceStore } from "@/stores/workspaceStore";

jest.mock("@/api/client", () => ({ api: { get: jest.fn(), post: jest.fn() } }));

const mockBack = jest.fn();
jest.mock("expo-router", () => ({
  useRouter: () => ({ back: mockBack }),
  useLocalSearchParams: () => ({ deployId: "d-1" }),
}));

let me: MyWorkspaceInfo;

const renderSheet = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <CancelDeploySheet />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  me = { role_name: "Developer", permissions: ["deploys:write"], show_locked_actions: false };
  useWorkspaceStore.setState({ selectedWorkspaceId: "ws-1" });
  mockBack.mockReset();
  jest.mocked(api.post).mockReset();
  jest.mocked(api.get).mockReset().mockImplementation(async (url: string) => {
    if (url === "/api/workspaces") return [{ id: "ws-1", name: "Acme" }];
    if (url === "/api/workspaces/ws-1/me") return me;
    if (url === "/api/deploys/d-1") return { id: "d-1", stack_id: "s-1", status: "running" };
    if (url === "/api/stacks/s-1") return { id: "s-1", project_id: "p-1", name: "checkout-api" };
    throw new Error(`unexpected GET ${url}`);
  });
});

describe("CancelDeploySheet", () => {
  test("a refused cancel shows the server's reason and keeps the sheet open", async () => {
    jest.mocked(api.post).mockRejectedValue(new ApiError(409, { message: "deploy d-1 is healthy, cannot cancel" }, "POST failed: 409"));
    await renderSheet();

    await userEvent.setup().press(await screen.findByRole("button", { name: "Cancel deploy" }));

    expect(await screen.findByText(/cannot cancel/)).toBeTruthy();
    expect(mockBack).not.toHaveBeenCalled();
  });

  test("names the stack it stops, posts the cancel and dismisses", async () => {
    jest.mocked(api.post).mockResolvedValue(null);
    await renderSheet();
    expect(await screen.findByText("Cancel the deploy of checkout-api?")).toBeTruthy();

    await userEvent.setup().press(await screen.findByRole("button", { name: "Cancel deploy" }));

    expect(api.post).toHaveBeenCalledWith("/api/deploys/d-1/cancel");
    expect(mockBack).toHaveBeenCalled();
  });

  test("Keep deploying dismisses without cancelling", async () => {
    await renderSheet();

    await userEvent.setup().press(screen.getByRole("button", { name: "Keep deploying" }));

    expect(api.post).not.toHaveBeenCalled();
    expect(mockBack).toHaveBeenCalled();
  });

  test.each([
    ["hides Cancel deploy from a role that can't deploy", { permissions: ["stacks:read"], show_locked_actions: false }, undefined],
    ["locks Cancel deploy for a role that shows locked actions", { permissions: ["stacks:read"], show_locked_actions: true }, "Your role can't cancel deploys"],
  ])("a deep link %s and sends nothing", async (_name, role, hint) => {
    me = { role_name: "Client", ...role };
    await renderSheet();

    await screen.findByText("Cancel the deploy of checkout-api?");
    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/api/workspaces/ws-1/me"));
    await act(async () => {});
    const control = screen.queryByRole("button", { name: "Cancel deploy" });
    expect(control === null).toBe(hint === undefined);
    expect(control?.props.accessibilityHint).toBe(hint);
    expect(api.post).not.toHaveBeenCalled();
  });
});

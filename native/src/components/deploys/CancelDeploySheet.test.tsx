import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, userEvent } from "@testing-library/react-native";

import { api } from "@/api/client";
import { ApiError } from "@/api/errors";
import { CancelDeploySheet } from "@/components/deploys/CancelDeploySheet";

jest.mock("@/api/client", () => ({ api: { get: jest.fn(), post: jest.fn() } }));

const mockBack = jest.fn();
jest.mock("expo-router", () => ({
  useRouter: () => ({ back: mockBack }),
  useLocalSearchParams: () => ({ deployId: "d-1" }),
}));

const renderSheet = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <CancelDeploySheet />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  mockBack.mockReset();
  jest.mocked(api.post).mockReset();
  jest.mocked(api.get).mockReset().mockImplementation(async (url: string) => {
    if (url === "/api/deploys/d-1") return { id: "d-1", stack_id: "s-1", status: "running" };
    if (url === "/api/stacks/s-1") return { id: "s-1", name: "checkout-api" };
    throw new Error(`unexpected GET ${url}`);
  });
});

describe("CancelDeploySheet", () => {
  test("a refused cancel shows the server's reason and keeps the sheet open", async () => {
    jest.mocked(api.post).mockRejectedValue(new ApiError(409, { message: "deploy d-1 is healthy, cannot cancel" }, "POST failed: 409"));
    await renderSheet();

    await userEvent.setup().press(screen.getByRole("button", { name: "Cancel deploy" }));

    expect(await screen.findByText(/cannot cancel/)).toBeTruthy();
    expect(mockBack).not.toHaveBeenCalled();
  });

  test("names the stack it stops, posts the cancel and dismisses", async () => {
    jest.mocked(api.post).mockResolvedValue(null);
    await renderSheet();
    expect(await screen.findByText("Cancel the deploy of checkout-api?")).toBeTruthy();

    await userEvent.setup().press(screen.getByRole("button", { name: "Cancel deploy" }));

    expect(api.post).toHaveBeenCalledWith("/api/deploys/d-1/cancel");
    expect(mockBack).toHaveBeenCalled();
  });

  test("Keep deploying dismisses without cancelling", async () => {
    await renderSheet();

    await userEvent.setup().press(screen.getByRole("button", { name: "Keep deploying" }));

    expect(api.post).not.toHaveBeenCalled();
    expect(mockBack).toHaveBeenCalled();
  });
});

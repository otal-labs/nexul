import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, userEvent } from "@testing-library/react-native";

import { api } from "@/api/client";
import { RedeployScreen } from "@/components/deploys/RedeployScreen";

jest.mock("@/api/client", () => ({
  api: { post: jest.fn() },
  errorMessage: jest.fn(() => "Something went wrong"),
}));

const mockBack = jest.fn();
jest.mock("expo-router", () => ({
  useRouter: () => ({ back: mockBack }),
  useLocalSearchParams: () => ({ stackId: "s-1", image: "api:2" }),
}));

const renderScreen = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <RedeployScreen />
    </QueryClientProvider>,
  );
};

beforeEach(() => {
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

    await userEvent.setup().press(screen.getByRole("button", { name: "Redeploy" }));

    expect(api.post).toHaveBeenCalledWith("/api/deploys", { stack_id: "s-1", image: "api:2" });
    expect(await screen.findByText("Redeploy")).toBeTruthy();
    expect(mockBack).toHaveBeenCalled();
  });

  test("Cancel dismisses the sheet without deploying", async () => {
    await renderScreen();

    await userEvent.setup().press(screen.getByRole("button", { name: "Cancel" }));

    expect(api.post).not.toHaveBeenCalled();
    expect(mockBack).toHaveBeenCalled();
  });
});

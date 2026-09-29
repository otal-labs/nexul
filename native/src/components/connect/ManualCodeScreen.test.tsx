import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, userEvent } from "@testing-library/react-native";

import { fetchAbout } from "@/api/connect";
import { ManualCodeScreen } from "@/components/connect/ManualCodeScreen";

jest.mock("@/api/connect", () => ({ fetchAbout: jest.fn(), exchangeConnectCode: jest.fn() }));
jest.mock("expo-router", () => ({
  useLocalSearchParams: () => ({ host: "https://old.example.com", code: "ABCD-EFGH-JKMN", auto: "1" }),
}));

describe("ManualCodeScreen", () => {
  test("a server too old to connect to offers another server instead of only retrying", async () => {
    jest.mocked(fetchAbout).mockResolvedValue({ product: "nexul", version: "v0.1.0" });
    const client = new QueryClient();
    await render(
      <QueryClientProvider client={client}>
        <ManualCodeScreen />
      </QueryClientProvider>,
    );

    await userEvent.setup().press(await screen.findByRole("button", { name: "Use a different server" }));

    expect(screen.getByDisplayValue("https://old.example.com")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Retry" })).toBeNull();
  });
});

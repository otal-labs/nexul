import { onlineManager } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react-native";

import { OfflineBanner } from "@/components/OfflineBanner";

jest.mock("react-native-safe-area-context", () => ({
  useSafeAreaInsets: () => ({ top: 0, right: 0, bottom: 0, left: 0 }),
}));

describe("OfflineBanner", () => {
  afterEach(() => onlineManager.setOnline(true));

  test("renders nothing while online", async () => {
    onlineManager.setOnline(true);
    await render(<OfflineBanner />);
    expect(screen.queryByRole("alert")).toBeNull();
  });

  test("shows the banner when the connection drops and hides it when it returns", async () => {
    await render(<OfflineBanner />);

    await act(async () => onlineManager.setOnline(false));
    expect(screen.getByRole("alert")).toBeTruthy();
    expect(screen.getByText(/Offline/)).toBeTruthy();

    await act(async () => onlineManager.setOnline(true));
    expect(screen.queryByRole("alert")).toBeNull();
  });
});

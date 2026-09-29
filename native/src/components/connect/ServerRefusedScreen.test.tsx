import { render, screen, userEvent } from "@testing-library/react-native";

import { ServerRefusedScreen } from "@/components/connect/ServerRefusedScreen";
import { MIN_SERVER_VERSION } from "@/lib/serverVersion";

describe("ServerRefusedScreen", () => {
  test("names the host, both versions, and offers Retry", async () => {
    const onRetry = jest.fn();
    await render(
      <ServerRefusedScreen host="https://nexul.example.com" version="v0.1.0" retrying={false} onRetry={onRetry}>
        {null}
      </ServerRefusedScreen>,
    );

    expect(
      screen.getByText(
        `https://nexul.example.com runs v0.1.0. This app needs ${MIN_SERVER_VERSION} or newer. Ask whoever runs it to upgrade.`,
      ),
    ).toBeTruthy();
    await userEvent.setup().press(screen.getByRole("button", { name: "Retry" }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });
});

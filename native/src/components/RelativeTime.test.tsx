import { act, render, screen } from "@testing-library/react-native";
import { Text } from "react-native";

import { RelativeTime } from "@/components/RelativeTime";
import { useClockStore } from "@/stores/clockStore";

const created = "2026-09-28T12:00:00Z";
const at = (iso: string) => Date.parse(iso);

test("a rendered age moves forward when the clock ticks", async () => {
  const now = jest.spyOn(Date, "now").mockReturnValue(at("2026-09-28T12:04:00Z"));
  await act(async () => useClockStore.getState().tick());
  await render(
    <Text>
      <RelativeTime iso={created} />
    </Text>,
  );
  expect(screen.getByText("4m")).toBeTruthy();

  now.mockReturnValue(at("2026-09-28T12:05:00Z"));
  await act(async () => useClockStore.getState().tick());

  expect(screen.getByText("5m")).toBeTruthy();
  now.mockRestore();
});

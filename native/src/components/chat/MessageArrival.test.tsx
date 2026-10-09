import { render, screen } from "@testing-library/react-native";
import { Text } from "react-native";

import { MessageArrival, type Arrival } from "@/components/chat/MessageArrival";
import { useEntrance } from "@/lib/motion";

jest.mock("@/lib/motion", () => {
  const actual = jest.requireActual<typeof import("@/lib/motion")>("@/lib/motion");
  return { ...actual, useEntrance: jest.fn(actual.useEntrance) };
});

// A thread opens on up to a screenful of rows at once, and only a message arriving while it is open moves.
test.each<[Arrival | undefined, number]>([
  [undefined, 0],
  ["arrive", 1],
  ["rise", 1],
])("a row arriving as %s sets up %i entrance", async (arrival, entrances) => {
  jest.mocked(useEntrance).mockClear();

  await render(
    <MessageArrival arrival={arrival}>
      <Text>hello</Text>
    </MessageArrival>,
  );

  expect(screen.getByText("hello")).toBeTruthy();
  expect(jest.mocked(useEntrance).mock.calls.length).toBe(entrances);
});

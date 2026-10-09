import { act, render } from "@testing-library/react-native";
import { createRef, useImperativeHandle, type Ref } from "react";

import { BoardDragContext, useBoardDragState, type BoardDrag } from "@/components/board/boardDrag";
import { TicketCard } from "@/components/board/TicketCard";
import { usePersonLookup } from "@/hooks/PeopleHooks";
import type { BoardStatus } from "@/models/Status";
import type { Ticket } from "@/models/Ticket";

// Every card body resolves its person once per render, so the lookup's call count is the cards' render count.
jest.mock("@/hooks/PeopleHooks", () => ({
  usePersonLookup: jest.fn(() => (login: string) => ({ user_id: login, login, display_name: login, avatar_url: "" })),
}));
jest.mock("@/hooks/WorkspaceHooks", () => ({ useCurrentWorkspaceId: () => "ws-1" }));

const statuses: BoardStatus[] = [
  { id: "st-todo", name: "Todo", position: 0, kind: "backlog" },
  { id: "st-done", name: "Done", position: 1, kind: "done" },
];
const tickets: Ticket[] = Array.from({ length: 20 }, (_, i) => ({
  id: `t-${i}`,
  project_id: "p-1",
  type_id: "",
  title: `Ticket ${i}`,
  status: "st-todo",
  position: i,
  number: i,
  developer: "",
  tester: "",
  labels: null,
})) as unknown as Ticket[];
const move = jest.fn();

const Board = ({ ref }: { ref: Ref<BoardDrag> }) => {
  const state = useBoardDragState(statuses, move);
  useImperativeHandle(ref, () => state, [state]);
  return (
    <BoardDragContext value={state}>
      {tickets.map((ticket) => (
        <TicketCard key={ticket.id} ticket={ticket} stage="backlog" projectPrefix="NEX" typeName={undefined} typeColor={undefined} onPress={move} />
      ))}
    </BoardDragContext>
  );
};

test("picking up, dropping and settling a card re-renders that card alone, not the board's other cards", async () => {
  const drag = createRef<BoardDrag>();
  await render(<Board ref={drag} />);
  const renders = jest.mocked(usePersonLookup);
  const mounted = renders.mock.calls.length;
  const from = { x: 0, y: 0, width: 300, height: 80 };
  const [held] = tickets;

  await act(async () => drag.current?.lift(held!, from));
  await act(async () => drag.current?.release(-1));
  await act(async () => drag.current?.clear());

  // The held card dims on lift and comes back on clear; the other 19 never render again.
  expect(renders.mock.calls.length - mounted).toBeLessThanOrEqual(2);
});

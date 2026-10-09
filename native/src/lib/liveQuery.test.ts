import { defineQuery } from "@/lib/liveQuery";

const fetch = () => Promise.resolve(null);

// `bun run typecheck` fails while any @ts-expect-error below stops erroring; the throw covers a definition cast past them.
test("a query cached until pushed cannot be declared without a topic that refreshes it", () => {
  // @ts-expect-error untilPushed needs at least one topic in refreshes
  expect(() => defineQuery({ key: "neverRefreshed", fetch, refreshes: {}, untilPushed: true })).toThrow(
    "neverRefreshed is cached until pushed but no topic refreshes it",
  );
});

// Never called: topic names and payload fields are checked against the event catalog at compile time only.
export const misdeclared = () => [
  // @ts-expect-error a topic the catalog does not publish
  defineQuery({ key: "misspelledTopic", fetch, refreshes: { "tickets.updated": "all" } }),
  // @ts-expect-error a field the topic's payload does not carry
  defineQuery({ key: "missingField", fetch, refreshes: { "ticket.deleted": { record: (p) => p.ticket_id } } }),
];

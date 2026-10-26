// @vitest-environment node
import { describe, expect, it } from "vitest";

import type { Person } from "@nexul/client-core/person";
import { findTicketPerson, namesAccount, ticketDeveloper, ticketTester } from "@nexul/client-core/ticketPeople";

const onik: Person = { user_id: "u-onik", login: "Onik97", display_name: "Onik", avatar_url: "/api/people/u-onik/avatar" };
const lena: Person = { user_id: "u-lena", login: "lena", display_name: "", avatar_url: "" };

describe("a ticket's people", () => {
  it("match by user id, and by login for someone who has not signed in", () => {
    const ticket = { developer: "Onik97", developer_id: "u-onik", tester: "sam", tester_id: "" };
    expect(ticketDeveloper(ticket)).toEqual({ key: "u-onik", login: "Onik97" });
    expect(ticketTester(ticket)).toEqual({ key: "sam", login: "sam" });
  });

  it("match by login when the server sends no ids", () => {
    expect(ticketDeveloper({ developer: "lena", tester: "" })).toEqual({ key: "lena", login: "lena" });
  });

  it.each([
    ["by id", { key: "u-onik", login: "onik97" }, true],
    ["by login from a server without ids", { key: "Onik97", login: "Onik97" }, true],
    ["not someone else", { key: "u-lena", login: "lena" }, false],
    ["not nobody", { key: "", login: "" }, false],
  ])("name the account %s", (_, ref, want) => {
    expect(namesAccount(ref, { id: "u-onik", login: "Onik97" })).toBe(want);
  });

  it.each([
    ["by id, whatever login the ticket carries", { key: "u-onik", login: "onik97" }, onik],
    ["by login from a server without ids", { key: "lena", login: "lena" }, lena],
    ["as the ticket's login when the directory has nobody", { key: "u-gone", login: "gone" }, { user_id: "gone", login: "gone", display_name: "", avatar_url: "" }],
  ])("are found %s", (_, ref, want) => {
    expect(findTicketPerson([onik, lena], ref)).toEqual(want);
  });
});

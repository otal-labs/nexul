import { describe, expect, it } from "vitest";

import { parseContainerPort } from "@/models/Stack";

describe("parseContainerPort", () => {
  it.each([
    ["8081:8080/tcp", { container: 8080, host: "8081", proto: "tcp" }],
    ["3000/tcp", { container: 3000, proto: "tcp" }],
    ["53/udp", { container: 53, proto: "udp" }],
    ["80:80", { container: 80, host: "80", proto: "tcp" }],
    ["127.0.0.1:5432:5432/tcp", { container: 5432, host: "127.0.0.1:5432", proto: "tcp" }],
  ])("reads %s", (entry, want) => {
    expect(parseContainerPort(entry)).toEqual(want);
  });

  it.each(["", "abc/tcp", "8080:/tcp", "0/tcp"])("rejects %j", (entry) => {
    expect(parseContainerPort(entry)).toBeUndefined();
  });
});

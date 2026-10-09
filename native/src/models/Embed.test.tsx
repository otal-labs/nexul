import { dropEchoedAuthor, embedCardTone, embedFold, embedFoldLabel, embedTimestamp, fieldTone, httpUrl, moreEmbedsLabel, type Embed, type EmbedTone } from "@/models/Embed";

const fields = (n: number) => Array.from({ length: n }, (_, i) => ({ name: `f${i}`, value: "v" }));

describe("embedFold", () => {
  test.each<[string, Embed, string | null]>([
    ["six fields fold nothing", { fields: fields(6) }, null],
    ["a seventh field folds one", { fields: fields(7) }, "Show 1 more field"],
    ["25 fields fold 19", { fields: fields(25) }, "Show 19 more fields"],
    ["a 420-character description stays open", { description: "x".repeat(420) }, null],
    ["a 421-character description clamps", { description: "x".repeat(421) }, "Show the rest"],
    ["eight short lines stay open", { description: Array(8).fill("a").join("\n") }, null],
    ["nine short lines clamp", { description: Array(9).fill("a").join("\n") }, "Show the rest"],
    ["hidden fields name the count even with a long description", { fields: fields(8), description: "x".repeat(500) }, "Show 2 more fields"],
  ])("%s", (_name, embed, label) => {
    expect(embedFoldLabel(embedFold(embed))).toBe(label);
  });

  test("names the embeds past two", () => {
    expect(moreEmbedsLabel(1)).toBe("Show 1 more embed");
    expect(moreEmbedsLabel(8)).toBe("Show 8 more embeds");
  });
});

// React Native's URL never rejects, so this pattern alone keeps a sender's script or data URL out of a link or image.
test.each<[string, string | undefined]>([
  ["javascript:alert(document.cookie)", undefined],
  ["data:image/png;base64,iVBORw0KGgo=", undefined],
  ["https://", undefined],
  ["https://example.com/a b", undefined],
  ["HTTPS://example.com/x.png", "HTTPS://example.com/x.png"],
  ["http://example.com", "http://example.com"],
])("httpUrl(%j) is %j", (raw, expected) => {
  expect(httpUrl(raw)).toBe(expected);
});

describe("embedTimestamp", () => {
  test("a timestamp without a zone reads as UTC, as Discord does", () => {
    expect(embedTimestamp("2026-10-06 21:11:00")).toBe("2026-10-06T21:11:00Z");
    expect(embedTimestamp("2026-10-06T21:11")).toBe("2026-10-06T21:11Z");
  });

  test("a timestamp with a zone is left alone", () => {
    expect(embedTimestamp("2026-10-06T21:11:00+02:00")).toBe("2026-10-06T21:11:00+02:00");
    expect(embedTimestamp("2026-10-06T21:11:00.000Z")).toBe("2026-10-06T21:11:00.000Z");
  });
});

describe("embedCardTone", () => {
  test.each<[string, Embed, EmbedTone | null]>([
    ["a failure in the title", { title: "atlas-api 0.4.11 failed its health check" }, "destructive"],
    ["the worst word wins", { title: "Deploy failed, rolled back to the healthy image" }, "destructive"],
    ["a healthy title", { title: "atlas-api 0.4.12 is healthy" }, "success"],
    ["an Uptime Kuma state", { title: "[Down] atlas-web" }, "destructive"],
    ["a title naming no state stays neutral even when the description does", { title: "Nightly deploy summary", description: "two redeployed" }, null],
    ["the description speaks when there is no title", { description: "Build passed in 3m" }, "success"],
    ["a word inside another word is not a state", { title: "Uploaded the backup" }, null],
  ])("%s", (_name, embed, tone) => {
    expect(embedCardTone(embed)).toBe(tone);
  });
});

describe("fieldTone", () => {
  test.each<[string, EmbedTone | null]>([
    ["healthy", "success"],
    ["timed out", "warning"],
    ["health check failed after 60s: connection refused on :8080", null],
  ])("%s", (value, tone) => {
    expect(fieldTone(value)).toBe(tone);
  });
});

describe("dropEchoedAuthor", () => {
  test("an author that only repeats the bot's name is dropped", () => {
    expect(dropEchoedAuthor({ title: "t", author: { name: "Deployer" } }, "Deployer").author).toBeUndefined();
  });

  test("an author with a link or icon stays", () => {
    const embed: Embed = { title: "t", author: { name: "Deployer", url: "https://example.com" } };
    expect(dropEchoedAuthor(embed, "Deployer")).toBe(embed);
  });
});

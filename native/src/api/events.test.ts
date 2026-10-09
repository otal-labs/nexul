import { buildLiveURL } from "@/api/events";

describe("buildLiveURL", () => {
  test("switches the scheme and carries the token as a query parameter", () => {
    expect(buildLiveURL("https://nexul.example.com", "ses_a+b")).toBe("wss://nexul.example.com/ws/events?token=ses_a%2Bb");
    expect(buildLiveURL("http://10.0.2.2:18980", "ses_x")).toBe("ws://10.0.2.2:18980/ws/events?token=ses_x");
  });
});

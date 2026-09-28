import { buildLiveURL, LiveEventsClient, type LiveSocket } from "@/api/events";

const fakeSocket = (): LiveSocket => ({
  onopen: null,
  onmessage: null,
  onclose: null,
  onerror: null,
  close: jest.fn(),
});

describe("LiveEventsClient", () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  test("drops a malformed frame and keeps delivering good ones", () => {
    const socket = fakeSocket();
    const client = new LiveEventsClient("ws://x", { wsFactory: () => socket });
    const handler = jest.fn();
    jest.spyOn(console, "warn").mockImplementation(() => undefined);
    client.subscribe(handler);
    client.connect();

    socket.onmessage?.({ data: "not json" });
    socket.onmessage?.({ data: '{"topic":"session.created","type":"event","payload":{}}' });

    expect(handler).toHaveBeenCalledTimes(1);
    expect(handler).toHaveBeenCalledWith({ topic: "session.created", type: "event", payload: {} });
  });

  test("reconnects after the socket drops", () => {
    const factory = jest.fn(fakeSocket);
    const client = new LiveEventsClient("ws://x", { wsFactory: factory });
    client.connect();

    factory.mock.results[0]?.value.onclose?.({});
    jest.runOnlyPendingTimers();

    expect(factory).toHaveBeenCalledTimes(2);
  });

  test("closing for the background never reconnects", () => {
    const factory = jest.fn(fakeSocket);
    const client = new LiveEventsClient("ws://x", { wsFactory: factory });
    client.connect();
    const socket = factory.mock.results[0]?.value as LiveSocket;

    client.close();
    socket.onclose?.({});
    jest.runOnlyPendingTimers();

    expect(socket.close).toHaveBeenCalled();
    expect(factory).toHaveBeenCalledTimes(1);
  });
});

describe("buildLiveURL", () => {
  test("switches the scheme and carries the token as a query parameter", () => {
    expect(buildLiveURL("https://nexul.example.com", "ses_a+b")).toBe("wss://nexul.example.com/ws/events?token=ses_a%2Bb");
    expect(buildLiveURL("http://10.0.2.2:18980", "ses_x")).toBe("ws://10.0.2.2:18980/ws/events?token=ses_x");
  });
});

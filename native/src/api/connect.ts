import { requestJSON } from "@/api/client";
import { ApiError } from "@/api/errors";
import type { About, ConnectDevice, ConnectExchange } from "@/models/Connect";

const isAbout = (body: unknown): body is About =>
  typeof body === "object" &&
  body !== null &&
  (body as About).product === "nexul" &&
  typeof (body as About).version === "string";

// Public on the server, so a typed address is confirmed as a Nexul server before a code is spent.
export const fetchAbout = async (host: string): Promise<About> => {
  let body: unknown;
  try {
    body = await requestJSON<unknown>(`${host}/api/about`);
  } catch (error) {
    if (error instanceof ApiError) throw new Error(`${host} is not a Nexul server.`);
    throw new Error(`Could not reach ${host}.`);
  }
  if (!isAbout(body)) throw new Error(`${host} is not a Nexul server.`);
  return body;
};

export const exchangeConnectCode = (host: string, code: string, device: ConnectDevice): Promise<ConnectExchange> =>
  requestJSON<ConnectExchange>(`${host}/api/auth/connect-codes/exchange`, {
    method: "POST",
    body: { code, device },
  });

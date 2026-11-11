import { createClient, type Client, type ResolveReferenceError } from "@1password/sdk";
import { hostVersion } from "./version.ts";
import { failed, type ResolveErrorKind, type ResolveResult, type Resolver } from "./resolver.ts";
import { log } from "./log.ts";

type Connect = (token: string) => Promise<Pick<Client, "secrets">>;

const connect: Connect = (token) => createClient({ auth: token, integrationName: "nexul", integrationVersion: hostVersion });

export function mapResolveError(error: ResolveReferenceError): ResolveErrorKind {
  switch (error.type) {
    case "itemNotFound":
      return "not_found";
    case "fieldNotFound":
      return "field_missing";
    case "vaultNotFound":
      return "no_access";
    default:
      return "other";
  }
}

// One client per token, created on first use and kept: creating one costs about a second of CPU, a read costs none.
export class OnePasswordResolver implements Resolver {
  readonly scheme = "op";
  private client: Promise<Pick<Client, "secrets">> | undefined;

  constructor(
    private readonly token: string,
    private readonly open: Connect = connect,
  ) {}

  async resolveAll(refs: string[]): Promise<Map<string, ResolveResult>> {
    const out = new Map<string, ResolveResult>();
    if (refs.length === 0) return out;
    try {
      this.client ??= this.open(this.token);
      const client = await this.client;
      const { individualResponses } = await client.secrets.resolveAll(refs);
      for (const ref of refs) {
        const response = individualResponses[ref];
        if (response?.content) {
          out.set(ref, { ok: true, value: response.content.secret });
          continue;
        }
        out.set(ref, failed(response?.error ? mapResolveError(response.error) : "other"));
      }
      return out;
    } catch (err) {
      this.client = undefined;
      log("warn", "1password unreachable", { error: err instanceof Error ? err.name : "unknown" });
      for (const ref of refs) out.set(ref, failed("unreachable"));
      return out;
    }
  }
}

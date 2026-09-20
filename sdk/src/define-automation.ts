import type { Topic, EventPayloads } from "./events.generated.ts";
import type { ConfigSchema } from "./config-schema.ts";
import type { Ctx } from "./context.ts";

export type Handler<T extends Topic, S extends ConfigSchema> = (payload: EventPayloads[T], ctx: Ctx<S>) => Promise<boolean>;

export interface DefineAutomationOptions<S extends ConfigSchema> {
  name: string;
  description: string;
  config?: S;
}

// Automation is the code-declared surface an automation announces over the
// dial-in connection: name, description, subscriptions, and config
// schema all come from here, never edited by the platform UI.
export class Automation<S extends ConfigSchema = ConfigSchema> {
  readonly name: string;
  readonly description: string;
  readonly configSchema: S;
  private readonly handlers = new Map<Topic, Handler<Topic, S>>();

  constructor(opts: DefineAutomationOptions<S>) {
    if (!opts.name.trim()) throw new Error("defineAutomation requires a non-empty name");
    this.name = opts.name;
    this.description = opts.description;
    this.configSchema = opts.config ?? ({} as S);
  }

  on<T extends Topic>(topic: T, handler: Handler<T, S>): this {
    this.handlers.set(topic, handler as Handler<Topic, S>);
    return this;
  }

  getHandler(topic: string): Handler<Topic, S> | undefined {
    return this.handlers.get(topic as Topic);
  }

  get subscriptions(): Topic[] {
    return [...this.handlers.keys()];
  }
}

export function defineAutomation<S extends ConfigSchema>(opts: DefineAutomationOptions<S>): Automation<S> {
  return new Automation(opts);
}

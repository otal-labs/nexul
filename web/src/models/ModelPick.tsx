import type { HarnessProvider, HarnessProviderModel, ModelOption, OptionSetting } from "@/models/Pairing";

// What a model picker holds: a provider instance id and a model slug, "" meaning the default at that level.
export interface ModelPick {
  provider: string;
  model: string;
}

export interface ModelRow {
  key: string;
  provider: HarnessProvider;
  model: HarnessProviderModel;
}

// The picker's left rail: every model, the starred ones, or one provider's.
export type RailFilter = { kind: "all" } | { kind: "favourites" } | { kind: "provider"; id: string };

// A favourite is keyed by the provider instance and the slug, since two providers can list the same slug.
export const modelKey = (providerId: string, slug: string) => `${providerId}::${slug}`;

export const findModel = (providers: HarnessProvider[], pick: ModelPick) =>
  providers.find((p) => p.id === pick.provider)?.models.find((m) => m.slug === pick.model);

// Every model the rail and the search let through, the current ones first and the legacy ones apart.
export const pickerRows = (
  providers: HarnessProvider[],
  filter: RailFilter,
  query: string,
  favourites: string[],
): { current: ModelRow[]; legacy: ModelRow[] } => {
  const terms = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
  const rows = providers
    .flatMap((provider) => provider.models.map((model) => ({ key: modelKey(provider.id, model.slug), provider, model })))
    .filter((row) => filter.kind !== "favourites" || favourites.includes(row.key))
    .filter((row) => filter.kind !== "provider" || row.provider.id === filter.id)
    .filter((row) => {
      const haystack = [row.model.name, row.model.slug, row.model.sub_provider ?? "", row.provider.name].join(" ").toLowerCase();
      return terms.every((t) => haystack.includes(t));
    });
  return { current: rows.filter((r) => !r.model.is_legacy), legacy: rows.filter((r) => r.model.is_legacy) };
};

// The provider line under a model's name: the provider, where a routed model comes from, and whether setup is still due.
export const sourceLabel = (row: ModelRow) =>
  [row.provider.name, row.model.sub_provider, row.provider.needs_setup && "needs setup"].filter(Boolean).join(" · ");

// One pickable row of the list: a model, or the default the picker offers first.
export interface PickerEntry {
  key: string;
  pick: ModelPick;
  name: string;
  source?: string;
  driver?: string;
  isNew?: boolean;
  favouriteKey?: string;
  legacy: boolean;
}

// The default row: the provider's own when one provider is in view, else the computer's.
const providerInView = (providers: HarnessProvider[], filter: RailFilter) => {
  if (filter.kind === "provider") return providers.find((p) => p.id === filter.id);
  if (providers.length === 1) return providers[0];
  return undefined;
};

const defaultEntry = (providers: HarnessProvider[], filter: RailFilter): PickerEntry => {
  const provider = providerInView(providers, filter);
  if (!provider) return { key: "default", pick: { provider: "", model: "" }, name: "Computer default", legacy: false };
  const fallback = provider.models.find((m) => m.is_default)?.name;
  return {
    key: `default::${provider.id}`,
    pick: { provider: provider.id, model: "" },
    name: "Provider default",
    ...(fallback && { source: fallback }),
    driver: provider.driver,
    legacy: false,
  };
};

// The rows in list order: the default when offered and nothing is searched, the current models, then the legacy ones.
export const pickerEntries = (
  providers: HarnessProvider[],
  filter: RailFilter,
  query: string,
  favourites: string[],
  allowDefault: boolean,
): PickerEntry[] => {
  const { current, legacy } = pickerRows(providers, filter, query, favourites);
  const toEntry = (row: ModelRow): PickerEntry => ({
    key: row.key,
    pick: { provider: row.provider.id, model: row.model.slug },
    name: row.model.name,
    source: sourceLabel(row),
    driver: row.provider.driver,
    ...(row.model.is_new && { isNew: true }),
    favouriteKey: row.key,
    legacy: !!row.model.is_legacy,
  });
  const offerDefault = allowDefault && !query.trim() && filter.kind !== "favourites";
  return [...(offerDefault ? [defaultEntry(providers, filter)] : []), ...current.map(toEntry), ...legacy.map(toEntry)];
};

export const optionDefault = (option: ModelOption): string | boolean | undefined =>
  option.type === "boolean" ? !!option.default_on : option.choices?.find((c) => c.is_default)?.id;

// What the option runs with: the setting when one is set, else the harness default.
export const optionValue = (option: ModelOption, settings: OptionSetting[]) =>
  settings.find((s) => s.id === option.id)?.value ?? optionDefault(option);

export const setOption = (settings: OptionSetting[], id: string, value: string | boolean): OptionSetting[] =>
  settings.some((s) => s.id === id)
    ? settings.map((s) => (s.id === id ? { id, value } : s))
    : [...settings, { id, value }];

// The options pill's text, such as "High · 1M": each select's choice, and a switch's label while it is on.
export const optionsLabel = (options: ModelOption[], settings: OptionSetting[]) =>
  options
    .flatMap((option) => {
      const value = optionValue(option, settings);
      if (option.type === "boolean") return value === true ? [option.label] : [];
      const choice = option.choices?.find((c) => c.id === value);
      return choice ? [choice.label] : [];
    })
    .join(" · ") || "Defaults";

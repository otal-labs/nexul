import { Stack, useLocalSearchParams, useRouter } from "expo-router";
import { useEffect } from "react";
import { ScrollView, View } from "react-native";

import { ClarifyProtoInbox } from "@/components/docs/prototype/ClarifyProtoInbox";
import { ClarifyProtoSwitcher } from "@/components/docs/prototype/ClarifyProtoSwitcher";
import { STATES, VARIANTS, useClarifyProtoStore, type ProtoVariant } from "@/components/docs/prototype/ClarifyProtoStore";
import { ClarifyProtoVariantA } from "@/components/docs/prototype/ClarifyProtoVariantA";
import { ClarifyProtoVariantB } from "@/components/docs/prototype/ClarifyProtoVariantB";
import { ClarifyProtoBottomBar, ClarifyProtoVariantC } from "@/components/docs/prototype/ClarifyProtoVariantC";
import { DOC_TITLE } from "@/components/docs/prototype/ClarifyProtoData";

const wrap = (i: number, n: number) => (i + n) % n;

// Throwaway: three ways a client answers a doc's questions on the phone, over mocked rounds, switched by ?variant=&state=.
export const ClarifyProtoScreen = () => {
  const router = useRouter();
  const params = useLocalSearchParams<{ variant?: string; state?: string; from?: string }>();
  const vi = Math.max(0, VARIANTS.findIndex((v) => v.key === params.variant));
  const variant: ProtoVariant = VARIANTS[vi]?.key ?? "A";
  const state = Math.min(STATES.length - 1, Math.max(0, Number(params.state ?? 2) || 0));
  const reset = useClarifyProtoStore((s) => s.reset);
  const shown = useClarifyProtoStore((s) => s.variant);

  const fromInbox = params.from === "inbox";

  // Arriving from the inbox lands on the questions themselves: B's first step, C's sheet.
  useEffect(() => {
    reset(state, variant);
    if (!fromInbox) return;
    if (variant === "B") useClarifyProtoStore.getState().setStep({ kind: "question", id: "r1q1" });
    if (variant !== "C") return;
    // ponytail: waits out the screen's own push, which swallows a second push started inside it.
    const timer = setTimeout(() => router.push("/more/docs/prototype-questions"), 450);
    return () => clearTimeout(timer);
  }, [reset, state, variant, fromInbox, router]);

  const go = (v: number, s: number) => router.setParams({ variant: VARIANTS[v]?.key ?? "A", state: String(s), from: "" });
  const inbox = state === 0;

  return (
    <View className="flex-1 bg-background">
      <Stack.Screen options={{ title: inbox ? "Inbox" : DOC_TITLE }} />
      <ScrollView className="flex-1" contentContainerClassName="p-4 pb-10" keyboardShouldPersistTaps="handled">
        <ClarifyProtoSwitcher
          variant={`${variant} · ${VARIANTS[vi]?.label}`}
          state={`${state} · ${STATES[state]}`}
          onVariant={(by) => go(wrap(vi + by, VARIANTS.length), state)}
          onState={(by) => go(vi, wrap(state + by, STATES.length))}
        />
        {inbox && <ClarifyProtoInbox onOpen={() => router.push({ pathname: "/more/docs/prototype", params: { variant, state: "1", from: "inbox" } })} />}
        {!inbox && shown === "A" && <ClarifyProtoVariantA />}
        {!inbox && shown === "B" && <ClarifyProtoVariantB />}
        {!inbox && shown === "C" && <ClarifyProtoVariantC />}
      </ScrollView>
      {!inbox && shown === "C" && <ClarifyProtoBottomBar />}
    </View>
  );
};

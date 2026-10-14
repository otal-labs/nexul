import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";

import { ShowcaseSurface } from "@/components/showcase/ShowcaseSurface";
import { demoInvite, worstInvite } from "@/proto/invitations/data";
import { AdminInviteForm } from "@/proto/invitations/AdminInviteForm";
import { ExistingAccountScreen, ExpiredScreen } from "@/proto/invitations/EndScreens";
import { RoleEditorDescription } from "@/proto/invitations/RoleEditorDescription";
import { VariantCards } from "@/proto/invitations/VariantCards";
import { VariantConsent } from "@/proto/invitations/VariantConsent";
import { VariantSplit } from "@/proto/invitations/VariantSplit";
import "@/proto/invitations/picker.css";

const VARIANTS = ["Consent", "Role cards", "Split"];
const SCREENS = ["before", "after", "expired", "existing", "admin", "role"];
const SCREEN_LABELS = ["Before sign-in", "After sign-in", "Expired", "Has account", "Admin form", "Role editor"];

const readParam = (name: string) => new URLSearchParams(window.location.search).get(name);

const writeParam = (name: string, value: string) => {
  const url = new URL(window.location.href);
  url.searchParams.set(name, value);
  window.history.replaceState(null, "", url);
};

interface PickerProps {
  label: string;
  items: string[];
  active: number;
  onSelect: (index: number) => void;
  position?: "top";
  extra?: { label: string; on: boolean; toggle: () => void };
}

const Picker = ({ label, items, active, onSelect, position, extra }: PickerProps) => {
  const nav = useRef<HTMLElement>(null);
  const highlight = useRef<HTMLSpanElement>(null);
  const [ready, setReady] = useState(false);
  useLayoutEffect(() => {
    const el = nav.current?.querySelectorAll<HTMLButtonElement>(".proto-picker-item:not(.proto-extra)")[active];
    if (!el || !highlight.current) return;
    highlight.current.style.width = `${el.offsetWidth}px`;
    highlight.current.style.transform = `translateX(${el.offsetLeft}px)`;
  }, [active]);
  useEffect(() => {
    const id = requestAnimationFrame(() => requestAnimationFrame(() => setReady(true)));
    return () => cancelAnimationFrame(id);
  }, []);
  return (
    <nav ref={nav} className="proto-picker" aria-label={label} data-position={position} data-ready={ready ? "" : undefined}>
      <span ref={highlight} className="proto-picker-highlight" aria-hidden="true" />
      {items.map((item, index) => (
        <button
          key={item}
          type="button"
          className="proto-picker-item"
          data-active={index === active ? "" : undefined}
          aria-current={index === active ? "true" : undefined}
          onClick={() => onSelect(index)}
        >
          {item}
        </button>
      ))}
      {extra && <span className="proto-picker-divider" aria-hidden="true" />}
      {extra && (
        <button type="button" className="proto-picker-item proto-extra" data-active={extra.on ? "" : undefined} onClick={extra.toggle}>
          {extra.label}
        </button>
      )}
    </nav>
  );
};

export const ProtoInvitationsPage = () => {
  const [variant, setVariant] = useState(() => Math.max(0, Math.min(VARIANTS.length - 1, Number(readParam("v") ?? 1) - 1)));
  const [screen, setScreen] = useState(() => Math.max(0, SCREENS.indexOf(readParam("s") ?? "before")));
  const [worst, setWorst] = useState(() => readParam("data") === "worst");
  const chrome = readParam("chrome") !== "0";
  const invite = worst ? worstInvite : demoInvite;
  const key = SCREENS[screen];
  const signedIn = key === "after";

  const pickVariant = useCallback((index: number) => {
    setVariant(index);
    writeParam("v", String(index + 1));
  }, []);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement;
      if (/^(INPUT|TEXTAREA|SELECT)$/.test(target.tagName) || target.isContentEditable) return;
      if (event.metaKey || event.ctrlKey || event.altKey) return;
      const num = Number.parseInt(event.key, 10);
      if (num >= 1 && num <= VARIANTS.length) pickVariant(num - 1);
      if (event.key === "ArrowRight") pickVariant((variant + 1) % VARIANTS.length);
      if (event.key === "ArrowLeft") pickVariant((variant - 1 + VARIANTS.length) % VARIANTS.length);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [pickVariant, variant]);

  const showcase = key === "before" || key === "after" || key === "expired" || key === "existing";

  return (
    <>
      {showcase && (
        <ShowcaseSurface>
          {(key === "before" || key === "after") && variant === 0 && <VariantConsent invite={invite} signedIn={signedIn} />}
          {(key === "before" || key === "after") && variant === 1 && <VariantCards invite={invite} signedIn={signedIn} />}
          {(key === "before" || key === "after") && variant === 2 && <VariantSplit invite={invite} signedIn={signedIn} />}
          {key === "expired" && <ExpiredScreen invite={invite} />}
          {key === "existing" && <ExistingAccountScreen invite={invite} />}
        </ShowcaseSurface>
      )}
      {key === "admin" && <AdminInviteForm worst={worst} />}
      {key === "role" && <RoleEditorDescription worst={worst} />}
      {chrome && (
        <Picker
          label="Screen"
          position="top"
          items={SCREEN_LABELS}
          active={screen}
          onSelect={(index) => {
            setScreen(index);
            writeParam("s", SCREENS[index] ?? "before");
          }}
          extra={{
            label: "Worst case",
            on: worst,
            toggle: () => {
              setWorst(!worst);
              writeParam("data", worst ? "demo" : "worst");
            },
          }}
        />
      )}
      {chrome && (key === "before" || key === "after") && <Picker label="Prototype variants" items={VARIANTS} active={variant} onSelect={pickVariant} />}
    </>
  );
};

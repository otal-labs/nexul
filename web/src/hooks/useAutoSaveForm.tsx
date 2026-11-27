import { useEffect, useRef } from "react";
import type { FieldValues, UseFormReturn } from "react-hook-form";

// Quiet gap after the last change, so a model pick that sets three fields is one write.
const SETTLE_MS = 200;

// Writes a form as its values change, with no Save: each settled change calls `save`, which returns what the server holds.
// A failed write puts the last saved values back (the caller's mutation says why) and a landed one calls `onSaved`.
export const useAutoSaveForm = <T extends FieldValues>(form: UseFormReturn<T>, save: (values: T) => Promise<T>, onSaved?: () => void) => {
  const last = useRef(JSON.stringify(form.getValues()));
  const timer = useRef<number | undefined>(undefined);
  const latest = useRef({ save, onSaved });
  useEffect(() => {
    latest.current = { save, onSaved };
  });

  useEffect(() => {
    const write = async () => {
      const values = form.getValues();
      if (JSON.stringify(values) === last.current) return;
      try {
        last.current = JSON.stringify(await latest.current.save(values));
        latest.current.onSaved?.();
      } catch {
        form.reset(JSON.parse(last.current) as T);
      }
    };
    const subscription = form.watch(() => {
      window.clearTimeout(timer.current);
      timer.current = window.setTimeout(() => void write(), SETTLE_MS);
    });
    return () => {
      subscription.unsubscribe();
      // Leaving the pane inside the gap still writes what was changed.
      if (timer.current !== undefined) {
        window.clearTimeout(timer.current);
        void write();
      }
    };
  }, [form]);
};

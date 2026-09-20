// Deliberately dumb: validation/probing/persistence all run in main; badges render probe state over IPC.
import { useCallback, useEffect, useState } from "react";

import type { ConnectionStatus, PublicVaultState } from "../electron/state";
import type { DesktopBridge } from "../electron/ipc";

const STATUS_LABEL: Record<ConnectionStatus, string> = {
  idle: "Not checked",
  connecting: "Connecting…",
  connected: "Connected",
  unreachable: "Unreachable",
  expired: "Token expired",
};

export interface AppProps {
  bridge?: DesktopBridge;
}

export const App = ({ bridge = window.desktop }: AppProps) => {
  const [state, setState] = useState<PublicVaultState | null>(null);
  const [token, setToken] = useState("");
  const [importing, setImporting] = useState(false);
  const [importError, setImportError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    const apply = (next: PublicVaultState) => {
      if (mounted) setState(next);
    };
    void bridge.listInstances().then(apply);
    const unsubscribe = bridge.onStateChanged(apply);
    return () => {
      mounted = false;
      unsubscribe();
    };
  }, [bridge]);

  const importToken = useCallback(async () => {
    setImporting(true);
    setImportError(null);
    try {
      const result = await bridge.importToken(token.trim());
      if (!result.ok) {
        setImportError(result.error ?? "Import failed");
        return;
      }
      setToken("");
    } catch (err) {
      setImportError(err instanceof Error ? err.message : "Import failed");
    } finally {
      setImporting(false);
    }
  }, [bridge, token]);

  const connect = useCallback((id: string) => void bridge.connect(id), [bridge]);
  const remove = useCallback((id: string) => void bridge.remove(id), [bridge]);
  const refresh = useCallback(() => void bridge.refresh(), [bridge]);

  const instances = state?.instances ?? [];
  const activeId = state?.activeId ?? null;

  return (
    <main className="launcher">
      <header className="launcher__header">
        <h1>Nexul</h1>
        <p className="launcher__subtitle">
          Connect to your instance. Paste a connection token generated in its workspace settings.
        </p>
      </header>

      <section className="launcher__card" aria-labelledby="import-title">
        <h2 id="import-title">Add instance</h2>
        <form
          className="launcher__form"
          onSubmit={(event) => {
            event.preventDefault();
            void importToken();
          }}
        >
          <input
            className="launcher__input"
            value={token}
            onChange={(event) => setToken(event.target.value)}
            placeholder="Paste connection token"
            aria-label="Connection token"
            spellCheck={false}
          />
          <button
            className="launcher__button"
            type="submit"
            disabled={importing || token.trim() === ""}
          >
            {importing ? "Importing…" : "Import"}
          </button>
        </form>
        {importError !== null && (
          <p className="launcher__error" role="alert">
            {importError}
          </p>
        )}
      </section>

      <section className="launcher__card" aria-labelledby="instances-title">
        <div className="launcher__card-head">
          <h2 id="instances-title">Instances</h2>
          <button className="launcher__button launcher__button--ghost" type="button" onClick={refresh}>
            Refresh
          </button>
        </div>
        {instances.length === 0 ? (
          <p className="launcher__empty">No instances yet — import a connection token above.</p>
        ) : (
          <ul className="launcher__list">
            {instances.map((instance) => (
              <li key={instance.id} className="launcher__row">
                <div className="launcher__row-main">
                  <p className="launcher__instance">
                    {instance.instanceUrl}
                    {instance.id === activeId && <span className="launcher__active">active</span>}
                  </p>
                  <p className="launcher__status">
                    <span className={`launcher__dot launcher__dot--${instance.connection.status}`} />
                    <span className="launcher__status-label">
                      {STATUS_LABEL[instance.connection.status]}
                    </span>
                    {instance.connection.error !== null && (
                      <span className="launcher__status-error">
                        {" — "}
                        {instance.connection.error}
                      </span>
                    )}
                    <span className="launcher__status-expiry">
                      {" · token expires "}
                      {new Date(instance.expiresAtSec * 1000).toLocaleDateString()}
                    </span>
                  </p>
                </div>
                <div className="launcher__row-actions">
                  <button
                    className="launcher__button"
                    type="button"
                    onClick={() => connect(instance.id)}
                    disabled={instance.connection.status === "connecting"}
                  >
                    Connect
                  </button>
                  <button
                    className="launcher__button launcher__button--danger"
                    type="button"
                    onClick={() => remove(instance.id)}
                  >
                    Remove
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </main>
  );
};

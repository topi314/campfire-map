const STORAGE_KEY = "campfire-export.wayfarer";

export interface WayfarerCredentials {
  enabled: boolean;
  session: string;
  xsrfToken: string;
}

function cleanCookieValue(raw: string) {
  return raw.trim().replace(/^["']|["']$/g, "");
}

export function defaultWayfarerCredentials(): WayfarerCredentials {
  return { enabled: false, session: "", xsrfToken: "" };
}

function sanitize(raw: unknown): WayfarerCredentials {
  const defaults = defaultWayfarerCredentials();
  if (!raw || typeof raw !== "object") return defaults;
  const obj = raw as Record<string, unknown>;
  return {
    enabled: typeof obj.enabled === "boolean" ? obj.enabled : defaults.enabled,
    session: typeof obj.session === "string" ? cleanCookieValue(obj.session) : "",
    xsrfToken: typeof obj.xsrfToken === "string" ? cleanCookieValue(obj.xsrfToken) : "",
  };
}

export function useWayfarerCredentials() {
  const enabled = useState("wayfarerEnabled", () => false);
  const session = useState("wayfarerSession", () => "");
  const xsrfToken = useState("wayfarerXsrf", () => "");

  onMounted(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return;
      const next = sanitize(JSON.parse(raw));
      enabled.value = next.enabled;
      session.value = next.session;
      xsrfToken.value = next.xsrfToken;
    } catch {
      /* private mode */
    }
  });

  const ready = computed(
    () =>
      enabled.value &&
      !!session.value.trim() &&
      !!xsrfToken.value.trim(),
  );

  function persist() {
    try {
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({
          enabled: enabled.value,
          session: session.value,
          xsrfToken: xsrfToken.value,
        } satisfies WayfarerCredentials),
      );
    } catch {
      /* private mode */
    }
  }

  function save(next: WayfarerCredentials) {
    const cleaned: WayfarerCredentials = {
      enabled: next.enabled,
      session: cleanCookieValue(next.session),
      xsrfToken: cleanCookieValue(next.xsrfToken),
    };
    if (cleaned.enabled && (!cleaned.session || !cleaned.xsrfToken)) {
      return false;
    }
    enabled.value = cleaned.enabled;
    session.value = cleaned.session;
    xsrfToken.value = cleaned.xsrfToken;
    persist();
    return true;
  }

  function clear() {
    enabled.value = false;
    session.value = "";
    xsrfToken.value = "";
    try {
      localStorage.removeItem(STORAGE_KEY);
    } catch {
      /* private mode */
    }
  }

  function invalidate() {
    clear();
  }

  function wayfarerHeaders(extra: Record<string, string> = {}) {
    if (!ready.value) return extra;
    return {
      ...extra,
      "X-Wayfarer-Session": session.value,
      "X-Wayfarer-XSRF": xsrfToken.value,
    };
  }

  return {
    enabled,
    session,
    xsrfToken,
    ready,
    save,
    clear,
    invalidate,
    wayfarerHeaders,
  };
}

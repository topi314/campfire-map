const STORAGE_KEY = "campfire-export.sessionToken";

export function normalizeSessionToken(raw: string) {
  return raw.trim().replace(/^Bearer\s+/i, "").replace(/^["']|["']$/g, "");
}

export function useSessionToken() {
  const token = useState("sessionToken", () => "");
  const settingsOpen = useState("settingsOpen", () => false);

  onMounted(() => {
    try {
      const saved = normalizeSessionToken(localStorage.getItem(STORAGE_KEY) || "");
      if (saved) {
        token.value = saved;
      }
    } catch {
      /* private mode */
    }
  });

  function save(next: string) {
    const cleaned = normalizeSessionToken(next);
    if (!cleaned) return false;
    token.value = cleaned;
    try {
      localStorage.setItem(STORAGE_KEY, cleaned);
    } catch {
      /* private mode */
    }
    settingsOpen.value = false;
    return true;
  }

  function openSettings() {
    settingsOpen.value = true;
  }

  function closeSettings() {
    settingsOpen.value = false;
  }

  function clearToken() {
    token.value = "";
    try {
      localStorage.removeItem(STORAGE_KEY);
    } catch {
      /* private mode */
    }
  }

  function invalidateToken() {
    clearToken();
    openSettings();
  }

  function authHeaders(extra: Record<string, string> = {}) {
    if (!token.value) return extra;
    return { ...extra, Authorization: `Bearer ${token.value}` };
  }

  return { token, settingsOpen, save, openSettings, closeSettings, clearToken, invalidateToken, authHeaders };
}

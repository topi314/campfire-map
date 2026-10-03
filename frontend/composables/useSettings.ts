const LEGACY_CAMPFIRE_TOKEN_KEY = "campfire-export.sessionToken";

export function useSettings() {
  const settingsOpen = useState("settingsOpen", () => false);

  onMounted(() => {
    try {
      localStorage.removeItem(LEGACY_CAMPFIRE_TOKEN_KEY);
    } catch {
      /* private mode */
    }
  });

  function openSettings() {
    settingsOpen.value = true;
  }

  function closeSettings() {
    settingsOpen.value = false;
  }

  return { settingsOpen, openSettings, closeSettings };
}

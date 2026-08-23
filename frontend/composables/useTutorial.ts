const STORAGE_KEY = "campfire-export.tutorialSeen";

export function useTutorial() {
  const tutorialOpen = useState("tutorialOpen", () => false);

  function hasSeenTutorial() {
    if (!import.meta.client) return true;
    return localStorage.getItem(STORAGE_KEY) === "1";
  }

  function markTutorialSeen() {
    if (!import.meta.client) return;
    try {
      localStorage.setItem(STORAGE_KEY, "1");
    } catch {
      /* private mode */
    }
  }

  function openTutorial() {
    tutorialOpen.value = true;
  }

  function closeTutorial() {
    tutorialOpen.value = false;
    markTutorialSeen();
  }

  function maybeShowTutorial() {
    if (!hasSeenTutorial()) {
      tutorialOpen.value = true;
    }
  }

  return { tutorialOpen, openTutorial, closeTutorial, maybeShowTutorial };
}

<template>
  <div
    v-if="open"
    class="token-overlay"
    role="dialog"
    aria-modal="true"
    aria-labelledby="settings-title"
    @click.self="onBackdropClick"
  >
    <form class="token-card export-card" @submit.prevent="submitActive">
      <div class="modal-header">
        <h2 id="settings-title">Settings</h2>
        <button type="button" class="modal-close" title="Close" aria-label="Close" @click="emit('close')">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path
              fill="currentColor"
              d="M18.3 5.71a1 1 0 0 0-1.41 0L12 10.59 7.11 5.7A1 1 0 0 0 5.7 7.11L10.59 12 5.7 16.89a1 1 0 1 0 1.41 1.41L12 13.41l4.89 4.89a1 1 0 0 0 1.41-1.41L13.41 12l4.89-4.89a1 1 0 0 0 0-1.4Z"
            />
          </svg>
        </button>
      </div>

      <div class="settings-section">
        <h3 class="settings-heading">Powerspot login</h3>
        <p class="export-sub">
          Only needed for Powerspots. Gyms, PokéStops, and routes load without login. Choose one source — credentials stay
          in this browser and are sent only to this app’s proxy.
        </p>

        <div class="auth-source-row" role="radiogroup" aria-label="Powerspot login source">
          <label class="filter">
            <input v-model="source" type="radio" value="campfire" />
            Campfire
            <span class="auth-source-hint">(active only)</span>
          </label>
          <label class="filter">
            <input v-model="source" type="radio" value="wayfarer" />
            Wayfarer
            <span class="auth-source-hint">(all)</span>
          </label>
        </div>

        <p class="counts settings-status">{{ statusText }}</p>

        <template v-if="source === 'campfire'">
          <details class="token-howto" :open="!token">
            <summary>How to get a Campfire token</summary>
            <div class="token-howto-body">
              <p class="token-howto-lead">
                1. Open
                <a href="https://campfire.scopely.com/discover" target="_blank" rel="noreferrer">campfire.scopely.com/discover</a>
                and sign in.
              </p>
              <p class="token-howto-lead">2. Follow the steps for your browser:</p>

              <div class="token-howto-browsers">
              <details class="token-howto-browser" :open="openBrowser === 'chrome'">
                <summary @click.prevent="toggleBrowser('chrome')">Chrome / Edge</summary>
                <ol class="tutorial-list token-howto-list">
                  <li>Right‑click the page → <strong>Inspect</strong>.</li>
                  <li>Open the <strong>Application</strong> tab.</li>
                  <li>
                    Left sidebar → <strong>Local Storage</strong> →
                    <code>https://campfire.scopely.com</code>.
                  </li>
                  <li>
                    Find <code>CapacitorStorage.sessionToken</code>, double‑click the value, and copy it.
                  </li>
                </ol>
              </details>

              <details class="token-howto-browser" :open="openBrowser === 'firefox'">
                <summary @click.prevent="toggleBrowser('firefox')">Firefox</summary>
                <ol class="tutorial-list token-howto-list">
                  <li>Right‑click the page → <strong>Inspect</strong>.</li>
                  <li>Open the <strong>Storage</strong> tab.</li>
                  <li>
                    Left sidebar → <strong>Local Storage</strong> →
                    <code>https://campfire.scopely.com</code>.
                  </li>
                  <li>
                    Find <code>CapacitorStorage.sessionToken</code>, double‑click the value, and copy it.
                  </li>
                </ol>
              </details>

              <details class="token-howto-browser" :open="openBrowser === 'safari'">
                <summary @click.prevent="toggleBrowser('safari')">Safari</summary>
                <ol class="tutorial-list token-howto-list">
                  <li>
                    Enable the Develop menu: <strong>Safari → Settings → Advanced</strong> → check
                    <strong>Show features for web developers</strong>.
                  </li>
                  <li><strong>Develop → Show Web Inspector</strong>.</li>
                  <li>Open the <strong>Storage</strong> tab.</li>
                  <li>
                    Left sidebar → <strong>Local Storage</strong> →
                    <code>https://campfire.scopely.com</code>.
                  </li>
                  <li>
                    Find <code>CapacitorStorage.sessionToken</code>, double‑click the value, and copy it.
                  </li>
                </ol>
              </details>
              </div>

              <p class="token-howto-lead">3. Paste the token below and click <strong>Save</strong>.</p>
              <p class="token-howto-note">
                If the key is missing, refresh Campfire while signed in and try again. Tokens expire — grab a new one
                if Powerspots stop loading.
              </p>
            </div>
          </details>

          <label class="token-label" for="session-token">Session token</label>
          <div class="secret-field">
            <input
              id="session-token"
              v-model="draft"
              class="search token-input"
              :type="showCampfireToken ? 'text' : 'password'"
              autocomplete="off"
              spellcheck="false"
              placeholder="Bearer eyJ… or the token only"
            />
            <button type="button" class="secret-toggle" @click="showCampfireToken = !showCampfireToken">
              {{ showCampfireToken ? "Hide" : "Show" }}
            </button>
          </div>
        </template>

        <template v-else>
          <details class="token-howto" :open="!wayReady">
            <summary>How to get SESSION and XSRF-TOKEN</summary>
            <div class="token-howto-body">
              <p class="token-howto-lead">
                1. Open
                <a href="https://wayfarer.scopely.com" target="_blank" rel="noreferrer">wayfarer.scopely.com</a>
                and sign in.
              </p>
              <p class="token-howto-lead">2. Follow the steps for your browser:</p>

              <div class="token-howto-browsers">
              <details class="token-howto-browser" :open="openBrowser === 'chrome'">
                <summary @click.prevent="toggleBrowser('chrome')">Chrome / Edge</summary>
                <ol class="tutorial-list token-howto-list">
                  <li>Right‑click the page → <strong>Inspect</strong>.</li>
                  <li>Open the <strong>Application</strong> tab.</li>
                  <li>
                    Left sidebar → <strong>Cookies</strong> →
                    <code>https://wayfarer.scopely.com</code>.
                  </li>
                  <li>
                    Copy the values for <code>SESSION</code> and <code>XSRF-TOKEN</code>.
                  </li>
                </ol>
              </details>

              <details class="token-howto-browser" :open="openBrowser === 'firefox'">
                <summary @click.prevent="toggleBrowser('firefox')">Firefox</summary>
                <ol class="tutorial-list token-howto-list">
                  <li>Right‑click the page → <strong>Inspect</strong>.</li>
                  <li>Open the <strong>Storage</strong> tab.</li>
                  <li>
                    Left sidebar → <strong>Cookies</strong> →
                    <code>https://wayfarer.scopely.com</code>.
                  </li>
                  <li>
                    Copy the values for <code>SESSION</code> and <code>XSRF-TOKEN</code>.
                  </li>
                </ol>
              </details>

              <details class="token-howto-browser" :open="openBrowser === 'safari'">
                <summary @click.prevent="toggleBrowser('safari')">Safari</summary>
                <ol class="tutorial-list token-howto-list">
                  <li>
                    Enable the Develop menu: <strong>Safari → Settings → Advanced</strong> → check
                    <strong>Show features for web developers</strong>.
                  </li>
                  <li><strong>Develop → Show Web Inspector</strong>.</li>
                  <li>Open the <strong>Storage</strong> tab.</li>
                  <li>
                    Left sidebar → <strong>Cookies</strong> →
                    <code>https://wayfarer.scopely.com</code>.
                  </li>
                  <li>
                    Copy the values for <code>SESSION</code> and <code>XSRF-TOKEN</code>.
                  </li>
                </ol>
              </details>
              </div>

              <p class="token-howto-lead">3. Paste both values below and click <strong>Save</strong>.</p>
              <p class="token-howto-note">Cookies expire — paste fresh ones if Powerspots stop loading.</p>
            </div>
          </details>

          <label class="token-label" for="wayfarer-session">SESSION</label>
          <div class="secret-field">
            <input
              id="wayfarer-session"
              v-model="waySession"
              class="search token-input"
              :type="showWaySession ? 'text' : 'password'"
              autocomplete="off"
              spellcheck="false"
              placeholder="SESSION cookie value"
            />
            <button type="button" class="secret-toggle" @click="showWaySession = !showWaySession">
              {{ showWaySession ? "Hide" : "Show" }}
            </button>
          </div>
          <label class="token-label" for="wayfarer-xsrf">XSRF-TOKEN</label>
          <div class="secret-field">
            <input
              id="wayfarer-xsrf"
              v-model="wayXsrf"
              class="search token-input"
              :type="showWayXsrf ? 'text' : 'password'"
              autocomplete="off"
              spellcheck="false"
              placeholder="XSRF-TOKEN cookie value"
            />
            <button type="button" class="secret-toggle" @click="showWayXsrf = !showWayXsrf">
              {{ showWayXsrf ? "Hide" : "Show" }}
            </button>
          </div>
        </template>

        <div class="btn-row" style="margin-top: 10px">
          <button type="submit" class="primary" :disabled="!canSave">Save</button>
          <button v-if="hasSavedForSource" type="button" @click="clearActive">Remove</button>
        </div>
      </div>

      <div class="settings-section">
        <h3 class="settings-heading">My Maps icons</h3>
        <p class="export-sub">
          Download PNG icons (gyms, stops, routes, etc.) to use as custom layer icons when styling your map in
          Google My Maps.
        </p>
        <button type="button" :disabled="downloadingIcons" @click="downloadIcons">
          {{ downloadingIcons ? "Preparing icons…" : "Download icon pack (ZIP)" }}
        </button>
      </div>

      <AppFooter />
    </form>
  </div>
</template>

<script setup lang="ts">
import { downloadMyMapsIconsZip } from "~/utils/myMapsIcons";

type AuthSource = "campfire" | "wayfarer";
type BrowserGuide = "chrome" | "firefox" | "safari";

const props = defineProps<{
  open: boolean;
  token: string;
  wayfarerEnabled: boolean;
  wayfarerSession: string;
  wayfarerXsrf: string;
}>();

const emit = defineEmits<{
  close: [];
  save: [token: string];
  clear: [];
  "save-wayfarer": [payload: { enabled: boolean; session: string; xsrfToken: string }];
  "clear-wayfarer": [];
}>();

const source = ref<AuthSource>("campfire");
const draft = ref("");
const waySession = ref("");
const wayXsrf = ref("");
const showCampfireToken = ref(false);
const showWaySession = ref(false);
const showWayXsrf = ref(false);
const downloadingIcons = ref(false);
const openBrowser = ref<BrowserGuide>("chrome");

function toggleBrowser(browser: BrowserGuide) {
  openBrowser.value = browser;
}

watch(source, () => {
  openBrowser.value = "chrome";
});

const wayDraftReady = computed(() => !!waySession.value.trim() && !!wayXsrf.value.trim());
const wayReady = computed(() => props.wayfarerEnabled && !!props.wayfarerSession && !!props.wayfarerXsrf);

const statusText = computed(() => {
  if (source.value === "campfire") {
    return props.token
      ? "Campfire token is saved."
      : "No Campfire token — paste one to load Powerspots.";
  }
  return wayReady.value
    ? "Wayfarer credentials are saved."
    : "No Wayfarer credentials — paste SESSION and XSRF-TOKEN to load Powerspots.";
});

const canSave = computed(() => {
  if (source.value === "campfire") return !!draft.value.trim();
  return wayDraftReady.value;
});

const hasSavedForSource = computed(() => {
  if (source.value === "campfire") return !!props.token;
  return wayReady.value || !!props.wayfarerSession || !!props.wayfarerXsrf;
});

watch(
  () =>
    [props.open, props.token, props.wayfarerEnabled, props.wayfarerSession, props.wayfarerXsrf] as const,
  ([open, token, wEnabled, wSession, wXsrf]) => {
    if (!open) return;
    draft.value = token;
    waySession.value = wSession;
    wayXsrf.value = wXsrf;
    if (wEnabled && wSession && wXsrf) {
      source.value = "wayfarer";
    } else if (token) {
      source.value = "campfire";
    } else if (wSession || wXsrf) {
      source.value = "wayfarer";
    } else {
      source.value = "campfire";
    }
  },
  { immediate: true },
);

function submitActive() {
  if (source.value === "campfire") {
    const next = normalizeSessionToken(draft.value);
    if (!next) return;
    emit("save", next);
    return;
  }
  if (!wayDraftReady.value) return;
  emit("save-wayfarer", {
    enabled: true,
    session: waySession.value,
    xsrfToken: wayXsrf.value,
  });
}

function onBackdropClick() {
  emit("close");
}

function clearActive() {
  if (source.value === "campfire") {
    draft.value = "";
    emit("clear");
    return;
  }
  waySession.value = "";
  wayXsrf.value = "";
  emit("clear-wayfarer");
}

async function downloadIcons() {
  downloadingIcons.value = true;
  try {
    await downloadMyMapsIconsZip();
  } catch (e) {
    alert(e instanceof Error ? e.message : "Failed to prepare icons");
  } finally {
    downloadingIcons.value = false;
  }
}
</script>

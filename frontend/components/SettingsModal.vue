<template>
  <div
    v-if="open"
    class="token-overlay"
    role="dialog"
    aria-modal="true"
    aria-labelledby="settings-title"
    @click.self="onBackdropClick"
  >
    <form class="token-card export-card" @submit.prevent="submit">
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
          Only needed for Powerspots. Gyms, PokéStops, and routes load without login. Credentials stay in this browser
          and are sent only to this app’s proxy.
        </p>

        <p class="counts settings-status">{{ statusText }}</p>

        <details class="token-howto" :open="!wayReady">
          <summary>
            <span class="token-howto-chevron" aria-hidden="true"></span>
            <span class="token-howto-summary-text">
              <span class="token-howto-title">How to get SESSION and XSRF-TOKEN</span>
              <span class="token-howto-sub">Copy them from Wayfarer cookies</span>
            </span>
          </summary>
          <div class="token-howto-body">
            <div class="token-howto-browsers" role="tablist" aria-label="Browser">
              <button
                type="button"
                role="tab"
                :class="{ active: openBrowser === 'chrome' }"
                :aria-selected="openBrowser === 'chrome'"
                @click="openBrowser = 'chrome'"
              >
                Chrome
              </button>
              <button
                type="button"
                role="tab"
                :class="{ active: openBrowser === 'edge' }"
                :aria-selected="openBrowser === 'edge'"
                @click="openBrowser = 'edge'"
              >
                Edge
              </button>
              <button
                type="button"
                role="tab"
                :class="{ active: openBrowser === 'firefox' }"
                :aria-selected="openBrowser === 'firefox'"
                @click="openBrowser = 'firefox'"
              >
                Firefox
              </button>
              <button
                type="button"
                role="tab"
                :class="{ active: openBrowser === 'safari' }"
                :aria-selected="openBrowser === 'safari'"
                @click="openBrowser = 'safari'"
              >
                Safari
              </button>
            </div>

            <ol v-if="openBrowser === 'chrome'" class="token-howto-steps">
              <li>
                <span class="token-howto-step">1</span>
                <span>
                  Open
                  <a href="https://wayfarer.scopely.com" target="_blank" rel="noreferrer"
                    >wayfarer.scopely.com</a
                  >
                  and sign in.
                </span>
              </li>
              <li>
                <span class="token-howto-step">2</span>
                <span>
                  Press <kbd>F12</kbd> (or right-click → Inspect) and open the
                  <strong>Application</strong> tab.
                </span>
              </li>
              <li>
                <span class="token-howto-step">3</span>
                <span>
                  In the left sidebar, expand <strong>Cookies</strong> and select
                  <code>https://wayfarer.scopely.com</code>.
                </span>
              </li>
              <li>
                <span class="token-howto-step">4</span>
                <span>
                  Copy the values for <code>SESSION</code> and <code>XSRF-TOKEN</code>.
                </span>
              </li>
              <li>
                <span class="token-howto-step">5</span>
                <span>Paste both values below and click <strong>Save</strong>.</span>
              </li>
            </ol>

            <ol v-else-if="openBrowser === 'edge'" class="token-howto-steps">
              <li>
                <span class="token-howto-step">1</span>
                <span>
                  Open
                  <a href="https://wayfarer.scopely.com" target="_blank" rel="noreferrer"
                    >wayfarer.scopely.com</a
                  >
                  and sign in.
                </span>
              </li>
              <li>
                <span class="token-howto-step">2</span>
                <span>
                  Press <kbd>F12</kbd> (or right-click → Inspect) and open the
                  <strong>Application</strong> tab. If you don’t see it, open the
                  <strong>≫</strong> menu in the DevTools toolbar.
                </span>
              </li>
              <li>
                <span class="token-howto-step">3</span>
                <span>
                  In the left sidebar, expand <strong>Cookies</strong> and select
                  <code>https://wayfarer.scopely.com</code>.
                </span>
              </li>
              <li>
                <span class="token-howto-step">4</span>
                <span>
                  Copy the values for <code>SESSION</code> and <code>XSRF-TOKEN</code>.
                </span>
              </li>
              <li>
                <span class="token-howto-step">5</span>
                <span>Paste both values below and click <strong>Save</strong>.</span>
              </li>
            </ol>

            <ol v-else-if="openBrowser === 'firefox'" class="token-howto-steps">
              <li>
                <span class="token-howto-step">1</span>
                <span>
                  Open
                  <a href="https://wayfarer.scopely.com" target="_blank" rel="noreferrer"
                    >wayfarer.scopely.com</a
                  >
                  and sign in.
                </span>
              </li>
              <li>
                <span class="token-howto-step">2</span>
                <span>
                  Press <kbd>F12</kbd> (or right-click → Inspect) and open the
                  <strong>Storage</strong> tab.
                </span>
              </li>
              <li>
                <span class="token-howto-step">3</span>
                <span>
                  In the left sidebar, expand <strong>Cookies</strong> and select
                  <code>https://wayfarer.scopely.com</code>.
                </span>
              </li>
              <li>
                <span class="token-howto-step">4</span>
                <span>
                  Copy the values for <code>SESSION</code> and <code>XSRF-TOKEN</code>.
                </span>
              </li>
              <li>
                <span class="token-howto-step">5</span>
                <span>Paste both values below and click <strong>Save</strong>.</span>
              </li>
            </ol>

            <ol v-else class="token-howto-steps">
              <li>
                <span class="token-howto-step">1</span>
                <span>
                  Open
                  <a href="https://wayfarer.scopely.com" target="_blank" rel="noreferrer"
                    >wayfarer.scopely.com</a
                  >
                  and sign in.
                </span>
              </li>
              <li>
                <span class="token-howto-step">2</span>
                <span>
                  Enable the Develop menu if needed:
                  <strong>Safari → Settings → Advanced → Show features for web developers</strong>.
                </span>
              </li>
              <li>
                <span class="token-howto-step">3</span>
                <span>
                  Choose <strong>Develop → Show Web Inspector</strong> (or
                  <kbd>⌥⌘I</kbd>), then open the <strong>Storage</strong> tab.
                </span>
              </li>
              <li>
                <span class="token-howto-step">4</span>
                <span>
                  Under <strong>Cookies</strong>, select
                  <code>https://wayfarer.scopely.com</code>, then copy
                  <code>SESSION</code> and <code>XSRF-TOKEN</code>.
                </span>
              </li>
              <li>
                <span class="token-howto-step">5</span>
                <span>Paste both values below and click <strong>Save</strong>.</span>
              </li>
            </ol>

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

        <div class="btn-row" style="margin-top: 10px">
          <button type="submit" class="primary" :disabled="!canSave">Save</button>
          <button v-if="hasSaved" type="button" @click="clear">Remove</button>
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

type BrowserGuide = "chrome" | "edge" | "firefox" | "safari";

const props = defineProps<{
  open: boolean;
  wayfarerEnabled: boolean;
  wayfarerSession: string;
  wayfarerXsrf: string;
}>();

const emit = defineEmits<{
  close: [];
  "save-wayfarer": [payload: { enabled: boolean; session: string; xsrfToken: string }];
  "clear-wayfarer": [];
}>();

const waySession = ref("");
const wayXsrf = ref("");
const showWaySession = ref(false);
const showWayXsrf = ref(false);
const downloadingIcons = ref(false);
const openBrowser = ref<BrowserGuide>("chrome");

const wayDraftReady = computed(() => !!waySession.value.trim() && !!wayXsrf.value.trim());
const wayReady = computed(() => props.wayfarerEnabled && !!props.wayfarerSession && !!props.wayfarerXsrf);

const statusText = computed(() =>
  wayReady.value
    ? "Wayfarer credentials are saved."
    : "No Wayfarer credentials — paste SESSION and XSRF-TOKEN to load Powerspots.",
);

const canSave = computed(() => wayDraftReady.value);

const hasSaved = computed(() => wayReady.value || !!props.wayfarerSession || !!props.wayfarerXsrf);

watch(
  () => [props.open, props.wayfarerEnabled, props.wayfarerSession, props.wayfarerXsrf] as const,
  ([open, , wSession, wXsrf]) => {
    if (!open) return;
    waySession.value = wSession;
    wayXsrf.value = wXsrf;
    openBrowser.value = "chrome";
  },
  { immediate: true },
);

function submit() {
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

function clear() {
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

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
        <h3 class="settings-heading">Campfire session token</h3>
        <p class="export-sub">
          Optional — only needed for Powerspots. Gyms, PokéStops, and routes load without a token. Stored only in this
          browser’s local storage and sent to this app’s proxy — not to Google.
        </p>
        <p class="counts settings-status">
          {{
            token
              ? "A token is saved in this browser."
              : "No session token — browsing without auth. Add one to load Powerspots."
          }}
        </p>

        <details class="token-howto" :open="!token">
          <summary>How to get a token</summary>
          <div class="token-howto-body">
            <ol class="tutorial-list token-howto-list">
              <li>
                Open
                <a href="https://campfire.scopely.com/discover" target="_blank" rel="noreferrer">campfire.scopely.com/discover</a>
                and sign in.
              </li>
              <li>
                On that Campfire page, right‑click anywhere → choose <strong>Inspect</strong>
                (Safari: turn on the Develop menu first under Safari → Settings → Advanced, then
                Develop → Show Web Inspector).
              </li>
              <li>
                In the panel that opens, click the tab named
                <strong>Application</strong> (Chrome / Edge) or <strong>Storage</strong> (Firefox / Safari).
              </li>
              <li>
                On the left, open <strong>Local Storage</strong>, then click
                <code>https://campfire.scopely.com</code>.
              </li>
              <li>
                In the list, find <code>CapacitorStorage.sessionToken</code>.
                Double‑click its value and copy it.
              </li>
              <li>Paste it in the box below and click <strong>Save token</strong>.</li>
            </ol>
            <p class="token-howto-note">
              If you don’t see that key, refresh the Campfire page while signed in and try again.
              Tokens expire — grab a new one if Powerspots stop loading.
            </p>
          </div>
        </details>

        <label class="token-label" for="session-token">Session token</label>
        <textarea
          id="session-token"
          v-model="draft"
          class="search token-input"
          rows="5"
          autocomplete="off"
          spellcheck="false"
          placeholder="Bearer eyJ… or the token only"
        />
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

      <div class="btn-row" style="margin-top: 14px">
        <button type="submit" class="primary" :disabled="!draft.trim()">Save token</button>
        <button v-if="token" type="button" @click="clearToken">Remove token</button>
      </div>

      <AppFooter />
    </form>
  </div>
</template>

<script setup lang="ts">
import { downloadMyMapsIconsZip } from "~/utils/myMapsIcons";

const props = defineProps<{
  open: boolean;
  token: string;
}>();

const emit = defineEmits<{
  close: [];
  save: [token: string];
  clear: [];
}>();

const draft = ref("");
const downloadingIcons = ref(false);

watch(
  () => [props.open, props.token] as const,
  ([open, token]) => {
    if (open) draft.value = token;
  },
  { immediate: true },
);

function submit() {
  const next = normalizeSessionToken(draft.value);
  if (!next) return;
  emit("save", next);
}

function onBackdropClick() {
  emit("close");
}

function clearToken() {
  draft.value = "";
  emit("clear");
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

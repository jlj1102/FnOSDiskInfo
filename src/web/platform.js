// fnOS micro-app bridge: keeps the menubar palette in sync with the host
// theme (JS SDK, developer.fnnas.com /api/platform-config/ + /api/page/ui/).
// Any failure (standalone browser, older fnOS, absent host) is ignored — the
// prefers-color-scheme fallback in style.css still applies.
import { TrimApp } from "./trim/index.js";

const POLL_MS = 3000;

function applyTheme(theme) {
  if (theme === "dark" || theme === "light") {
    document.documentElement.dataset.navbar = theme;
  }
}

try {
  const sdk = new TrimApp();
  // getPlatformConfig goes through the host handshake, so isStandaloneWeb is
  // settled afterwards (the SDK: standalone === window.parent === window).
  const config = await sdk.getPlatformConfig();
  if (sdk.isStandaloneWeb !== true) {
    applyTheme(config && config.theme);
    if (sdk.isWeb === true) {
      try {
        await sdk.$on("os/theme", applyTheme);
      } catch (e) {
        // push not available; the poll below covers it
      }
    }
    // fnOS does not always push os/theme to third-party pages: poll the host
    // so switching light/dark takes effect without reloading the app.
    let busy = false;
    setInterval(async () => {
      if (busy || document.hidden) {
        return;
      }
      busy = true;
      try {
        const cfg = await sdk.getPlatformConfig();
        applyTheme(cfg && cfg.theme);
      } catch (e) {
        // keep the last known theme
      } finally {
        busy = false;
      }
    }, POLL_MS);
  }
} catch (e) {
  // not running inside the fnOS host: keep the CSS fallback
}

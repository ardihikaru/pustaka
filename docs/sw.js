// Service-worker kill switch. Keep this public even while auth is enabled so a
// browser can replace an obsolete cached worker after the site becomes private.
self.addEventListener("install", () => self.skipWaiting());
self.addEventListener("activate", event => event.waitUntil(self.registration.unregister()));

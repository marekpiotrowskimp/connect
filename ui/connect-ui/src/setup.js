// Connect Tauri bridge — loaded before app.js
(function () {
  window.CONNECT = window.CONNECT || {};

  window.CONNECT.isNative = typeof window.__TAURI__ !== "undefined";

  // Detect connectd port — store promise so load() can await it
  window.CONNECT.apiBase = window.CONNECT.isNative ? "http://127.0.0.1:47810" : "";
  window.CONNECT.portReady = (async function detectPort() {
    if (!window.CONNECT.isNative) return;
    try {
      var port = await window.__TAURI__.invoke("get_service_port");
      window.CONNECT.apiBase = "http://127.0.0.1:" + port;
    } catch (_) {
      // keep default port
    }
  })();

  // Forward deep-link events from Tauri eval to DOM custom event
  window.addEventListener("deep-link", function (event) {
    const url = typeof event.detail === "string" ? event.detail : "";
    if (!url.startsWith("connect://join/")) return;

    // Wait for DOM ready then pre-fill the pairing form
    function fill() {
      var input = document.getElementById("pair-url");
      var dlg = document.getElementById("connection-dialog");
      if (input && dlg) {
        input.value = url.replace("connect://", "http://");
        if (!dlg.open) dlg.showModal();
        return true;
      }
      return false;
    }
    if (!fill()) {
      var iv = setInterval(function () {
        if (fill()) clearInterval(iv);
      }, 100);
      setTimeout(function () { clearInterval(iv); }, 10000);
    }
  });
})();

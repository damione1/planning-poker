// Room creation form
Alpine.data("roomForm", () => ({
  pointingMethod: "fibonacci",
}));

// Card selector
Alpine.data("cardSelector", () => ({
  selected: null,

  selectCard(value) {
    this.selected = value;

    // Send vote via WebSocket
    const message = JSON.stringify({
      type: "vote",
      value: value,
    });

    // htmx-ext-ws handles sending
    const wsElement = document.querySelector("[ws-connect]");
    if (wsElement) {
      // Use htmx's ws-send mechanism
      htmx.trigger(wsElement, "ws-send", { value: message });
    }
  },
}));

// Room sharing
Alpine.data("roomSharing", () => ({
  showQR: false,
  copied: false,

  async copyUrl() {
    try {
      await navigator.clipboard.writeText(window.location.href);
      this.copied = true;
      setTimeout(() => {
        this.copied = false;
      }, 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  },
}));

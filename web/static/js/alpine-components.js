// Register components directly - Alpine is already loaded due to defer ordering
console.log('Registering Alpine components...');

// Room creation form
Alpine.data("roomForm", () => ({
  pointingMethod: "fibonacci",
}));

// Room state management
Alpine.data("roomState", () => ({
    ws: null,
    roomId: null,

    init() {
      console.log('[DEBUG] roomState init() called');
      // Extract room ID from ws-connect attribute
      const wsElement = document.querySelector("[ws-connect]");
      console.log('[DEBUG] wsElement found:', wsElement);
      if (wsElement) {
        const wsUrl = wsElement.getAttribute("ws-connect");
        console.log('[DEBUG] wsUrl:', wsUrl);
        this.roomId = wsUrl.replace("/ws/", "");
        console.log('[DEBUG] Room ID extracted:', this.roomId);

        // Connect to WebSocket directly
        this.connectWebSocket(wsUrl);
      } else {
        console.error('[ERROR] ws-connect element not found!');
      }
    },

    connectWebSocket(path) {
      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      const wsUrl = `${protocol}//${window.location.host}${path}`;

      console.log("Connecting to WebSocket:", wsUrl);

      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        console.log("✓ WebSocket connected");
      };

      this.ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          console.log("WebSocket message received:", message);
          this.handleMessage(message);
        } catch (err) {
          console.error("Failed to parse WebSocket message:", err);
        }
      };

      this.ws.onerror = (error) => {
        console.error("WebSocket error:", error);
      };

      this.ws.onclose = () => {
        console.log("WebSocket closed");
        // Attempt reconnect after 3 seconds
        setTimeout(() => {
          console.log("Attempting to reconnect...");
          this.connectWebSocket(path);
        }, 3000);
      };
    },

    handleMessage(message) {
      console.log("WebSocket message received:", message);

      switch (message.type) {
        case "vote_cast":
          this.handleVoteCast(message.payload);
          break;
        case "votes_revealed":
          this.handleVotesRevealed(message.payload);
          break;
        case "room_reset":
          this.handleRoomReset();
          break;
        case "participant_joined":
          this.handleParticipantJoined(message.payload);
          break;
        case "participant_left":
          this.handleParticipantLeft(message.payload);
          break;
      }
    },

    handleVoteCast(payload) {
      // Reload participant grid to show vote status
      window.location.reload();
    },

    handleVotesRevealed(payload) {
      // Reload to show revealed votes
      window.location.reload();
    },

    handleRoomReset() {
      // Reload to reset UI
      window.location.reload();
    },

    handleParticipantJoined(payload) {
      // Reload to show new participant
      window.location.reload();
    },

    handleParticipantLeft(payload) {
      // Reload to remove participant
      window.location.reload();
    },
  }));

  // Card selector
  Alpine.data("cardSelector", () => ({
    selected: null,

    selectCard(value) {
      this.selected = value;
      console.log('Card selected:', value);

      // Send vote via WebSocket
      const message = {
        type: "vote",
        payload: {
          value: value,
        },
      };

      // Get WebSocket from roomState component
      const roomState = Alpine.$data(document.querySelector('[x-data*="roomState"]'));
      if (roomState && roomState.ws && roomState.ws.readyState === WebSocket.OPEN) {
        roomState.ws.send(JSON.stringify(message));
        console.log('Vote sent via WebSocket:', message);
      } else {
        console.error('WebSocket not connected - cannot send vote');
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

console.log('✓ All Alpine components registered');

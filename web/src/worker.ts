// SharedWorker for coordinating multiple tabs
// CRC: crc-SharedWorker.md
// Spec: interfaces.md

interface WorkerMessage {
  type: string;
  data?: unknown;
}

interface TabConnection {
  port: MessagePort;
  sessionId: string;
  isMain: boolean;
}

const tabs: Map<MessagePort, TabConnection> = new Map();
const sessions: Map<string, WebSocket> = new Map();

// Handle new tab connections
// Using `any` because SharedWorkerGlobalScope types aren't included by default
// eslint-disable-next-line @typescript-eslint/no-explicit-any
(self as any).onconnect = (event: MessageEvent) => {
  const port = event.ports[0];

  port.onmessage = (e: MessageEvent<WorkerMessage>) => {
    handleTabMessage(port, e.data);
  };

  port.start();
};

function handleTabMessage(port: MessagePort, msg: WorkerMessage): void {
  switch (msg.type) {
    case 'register':
      handleRegister(port, msg.data as { sessionId: string });
      break;
    case 'send':
      handleSend(port, msg.data);
      break;
    case 'unregister':
      handleUnregister(port);
      break;
  }
}

function handleRegister(port: MessagePort, data: { sessionId: string }): void {
  const { sessionId } = data;

  // Check if we already have a connection for this session
  const existingWs = sessions.get(sessionId);
  const isMain = !existingWs;

  tabs.set(port, {
    port,
    sessionId,
    isMain,
  });

  if (isMain) {
    // Create WebSocket connection for this session
    const ws = createWebSocket(sessionId);
    sessions.set(sessionId, ws);
  }

  // Notify tab of its role
  port.postMessage({
    type: 'registered',
    data: { isMain },
  });
}

function handleSend(port: MessagePort, data: unknown): void {
  const tab = tabs.get(port);
  if (!tab) return;

  const ws = sessions.get(tab.sessionId);
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(data));
  }
}

function handleUnregister(port: MessagePort): void {
  const tab = tabs.get(port);
  if (!tab) return;

  tabs.delete(port);

  // Check if this was the last tab for this session
  const remainingTabs = Array.from(tabs.values()).filter((t) => t.sessionId === tab.sessionId);

  if (remainingTabs.length === 0) {
    // Close WebSocket
    const ws = sessions.get(tab.sessionId);
    if (ws) {
      ws.close();
      sessions.delete(tab.sessionId);
    }
  } else if (tab.isMain) {
    // Promote another tab to main
    const newMain = remainingTabs[0];
    newMain.isMain = true;
    newMain.port.postMessage({ type: 'promoted', data: { isMain: true } });
  }
}

function createWebSocket(sessionId: string): WebSocket {
  const protocol = self.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const ws = new WebSocket(`${protocol}//${self.location.host}/ws/${sessionId}`);

  ws.onmessage = (event) => {
    // Broadcast to all tabs for this session
    const message = JSON.parse(event.data);
    broadcastToSession(sessionId, { type: 'message', data: message });
  };

  ws.onclose = () => {
    broadcastToSession(sessionId, { type: 'disconnected' });
    sessions.delete(sessionId);
  };

  ws.onerror = () => {
    broadcastToSession(sessionId, { type: 'error', data: 'WebSocket error' });
  };

  return ws;
}

function broadcastToSession(sessionId: string, msg: WorkerMessage): void {
  tabs.forEach((tab) => {
    if (tab.sessionId === sessionId) {
      tab.port.postMessage(msg);
    }
  });
}

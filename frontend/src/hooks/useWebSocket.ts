import { useState, useEffect, useRef, useCallback } from 'react';

// Define expected message structure (adjust based on backend ticker.PriceUpdate)
interface PriceUpdateMessage {
  symbol: string;
  price: number;
  ts: number;
}

interface WebSocketHookOptions {
  url: string;
  onMessage: (data: PriceUpdateMessage) => void;
  onError?: (event: Event) => void;
  onOpen?: (event: Event) => void;
  onClose?: (event: CloseEvent) => void;
  retryInterval?: number; // Time in ms to wait before retrying connection
}

function useWebSocket({
  url,
  onMessage,
  onError,
  onOpen,
  onClose,
  retryInterval = 5000, // Default retry interval: 5 seconds
}: WebSocketHookOptions) {
  const ws = useRef<WebSocket | null>(null);
  const retryTimeout = useRef<NodeJS.Timeout | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [shouldConnect, setShouldConnect] = useState(true); // Control connection attempts

  const connect = useCallback(() => {
    if (!url || !shouldConnect) return;

    console.log(`WebSocket: Attempting to connect to ${url}...`);
    ws.current = new WebSocket(url);

    ws.current.onopen = (event) => {
      console.log(`WebSocket: Connection opened to ${url}`);
      setIsConnected(true);
      if (retryTimeout.current) {
        clearTimeout(retryTimeout.current);
        retryTimeout.current = null;
      }
      onOpen?.(event);
    };

    ws.current.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        // Basic validation (can be more robust)
        if (typeof data === 'object' && data !== null && 'symbol' in data && 'price' in data) {
          onMessage(data as PriceUpdateMessage);
        } else {
          console.warn('WebSocket: Received non-price update message:', data);
        }
      } catch (error) {
        console.error('WebSocket: Error parsing message:', error, event.data);
      }
    };

    ws.current.onerror = (event) => {
      console.error('WebSocket: Error:', event);
      setIsConnected(false);
      onError?.(event);
      // Don't retry immediately here, onClose handles retry logic
    };

    ws.current.onclose = (event) => {
      console.log(`WebSocket: Connection closed to ${url}`, event.code, event.reason);
      setIsConnected(false);
      ws.current = null; // Clear the ref
      onClose?.(event);

      // Schedule retry if connection should still be active
      if (shouldConnect) {
          console.log(`WebSocket: Scheduling reconnect in ${retryInterval}ms...`);
          if (retryTimeout.current) clearTimeout(retryTimeout.current); // Clear previous timeout if any
          retryTimeout.current = setTimeout(connect, retryInterval);
      }
    };

  }, [url, onMessage, onError, onOpen, onClose, retryInterval, shouldConnect]);

  const disconnect = useCallback(() => {
      setShouldConnect(false); // Prevent automatic retries
      if (retryTimeout.current) {
          clearTimeout(retryTimeout.current);
          retryTimeout.current = null;
      }
      if (ws.current) {
          console.log(`WebSocket: Manually disconnecting from ${url}...`);
          ws.current.close(1000, 'Client disconnected'); // 1000: Normal Closure
          ws.current = null;
      }
      setIsConnected(false);
  }, [url]);

  useEffect(() => {
      setShouldConnect(true); // Enable connection attempts when url changes or component mounts
      connect();

      // Cleanup function on component unmount or URL change
      return () => {
        disconnect();
      };
  }, [connect, disconnect]); // Re-run if connect/disconnect functions change (due to dependency changes)

  // Function to manually send messages (optional)
  const sendMessage = useCallback((message: string) => {
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      ws.current.send(message);
    } else {
      console.error('WebSocket: Cannot send message, connection not open.');
    }
  }, []);

  return { isConnected, sendMessage };
}

export default useWebSocket; 
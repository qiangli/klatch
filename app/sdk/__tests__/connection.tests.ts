import { Connection } from '../src/connection';
import { ConsoleLogging } from '../src/logging';

class FakeWebSocket {
  static sockets: FakeWebSocket[] = [];
  onclose: (() => void) | null = null;
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  readyState = 0;
  constructor(public url: string) {
    FakeWebSocket.sockets.push(this);
  }
  send(_message: string) {}
  close() { this.onclose?.(); }
}

test('prefixed status socket backs off failed upgrades and stops after close', async () => {
  jest.useFakeTimers();
  jest.spyOn(Math, 'random').mockReturnValue(0);
  const original = global.WebSocket;
  global.WebSocket = FakeWebSocket as unknown as typeof WebSocket;
  try {
    const connection = new Connection(new ConsoleLogging(), 'token', 'sat.example/prefix/', true);
    expect(FakeWebSocket.sockets[0].url).toBe('wss://sat.example/prefix/status?mode=ring');
    FakeWebSocket.sockets[0].close(); // failed upgrade (including HTTP 429)
    jest.advanceTimersByTime(999);
    expect(FakeWebSocket.sockets).toHaveLength(1);
    jest.advanceTimersByTime(1);
    expect(FakeWebSocket.sockets).toHaveLength(2);
    FakeWebSocket.sockets[1].close();
    jest.advanceTimersByTime(1999);
    expect(FakeWebSocket.sockets).toHaveLength(2);
    await connection.close();
    jest.advanceTimersByTime(60000);
    expect(FakeWebSocket.sockets).toHaveLength(2);
  } finally {
    global.WebSocket = original;
    jest.restoreAllMocks();
    jest.useRealTimers();
    FakeWebSocket.sockets = [];
  }
});

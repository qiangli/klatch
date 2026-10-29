import { fetchWithTimeout, retryConfig } from '../src/net/fetchUtil';

const res = (status: number, retryAfter?: string) =>
  ({ status, headers: { get: (k: string) => (k === 'Retry-After' ? retryAfter ?? null : null) } }) as unknown as Response;

describe('fetchWithTimeout', () => {
  const orig = { ...retryConfig };
  beforeEach(() => {
    retryConfig.baseBackoffMs = 1;
  });
  afterEach(() => {
    Object.assign(retryConfig, orig);
    jest.restoreAllMocks();
  });

  test('retries on 429 honouring Retry-After', async () => {
    const f = jest.fn().mockResolvedValueOnce(res(429, '0')).mockResolvedValueOnce(res(429)).mockResolvedValueOnce(res(200));
    global.fetch = f as unknown as typeof fetch;
    const r = await fetchWithTimeout('http://x/a', { method: 'GET' });
    expect(r.status).toBe(200);
    expect(f).toHaveBeenCalledTimes(3);
  });

  test('waits for Retry-After seconds', async () => {
    const f = jest.fn().mockResolvedValueOnce(res(429, '1')).mockResolvedValueOnce(res(200));
    global.fetch = f as unknown as typeof fetch;
    const start = Date.now();
    await fetchWithTimeout('http://x/a', { method: 'POST' });
    expect(Date.now() - start).toBeGreaterThanOrEqual(950);
    expect(f).toHaveBeenCalledTimes(2);
  });

  test('gives up after max attempts and returns the 429', async () => {
    const f = jest.fn().mockResolvedValue(res(429, '0'));
    global.fetch = f as unknown as typeof fetch;
    const r = await fetchWithTimeout('http://x/a', { method: 'GET' });
    expect(r.status).toBe(429);
    expect(f).toHaveBeenCalledTimes(retryConfig.maxAttempts);
  });

  test('retries timeouts for GET', async () => {
    const f = jest.fn().mockImplementationOnce(() => new Promise(() => {})).mockResolvedValueOnce(res(200));
    global.fetch = f as unknown as typeof fetch;
    const r = await fetchWithTimeout('http://x/a', { method: 'GET' }, 20);
    expect(r.status).toBe(200);
    expect(f).toHaveBeenCalledTimes(2);
  });

  test('does not retry timeouts for POST', async () => {
    const f = jest.fn().mockImplementation(() => new Promise(() => {}));
    global.fetch = f as unknown as typeof fetch;
    await expect(fetchWithTimeout('http://x/a', { method: 'POST' }, 20)).rejects.toThrow('timeout');
    expect(f).toHaveBeenCalledTimes(1);
  });

  test('timeout is bounded', async () => {
    const f = jest.fn().mockImplementation(() => new Promise(() => {}));
    global.fetch = f as unknown as typeof fetch;
    await expect(fetchWithTimeout('http://x/a', { method: 'GET' }, 10)).rejects.toThrow('timeout');
    expect(f).toHaveBeenCalledTimes(retryConfig.maxAttempts);
  });
});

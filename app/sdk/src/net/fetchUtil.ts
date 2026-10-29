const TIMEOUT = 30000;
const MAX_ATTEMPTS = 4;
const MAX_RETRY_AFTER_MS = 30000;
const BASE_BACKOFF_MS = 1000;
const IDEMPOTENT = new Set(['GET', 'HEAD', 'PUT', 'DELETE', 'OPTIONS']);

export const retryConfig = { timeout: TIMEOUT, maxAttempts: MAX_ATTEMPTS, baseBackoffMs: BASE_BACKOFF_MS, maxRetryAfterMs: MAX_RETRY_AFTER_MS };

export function checkResponse(code: number) {
  if (code >= 400 && code < 600) {
    throw new Error(code.toString());
  }
}

function retryAfterMs(res: Response): number | null {
  const header = res.headers && typeof res.headers.get === 'function' ? res.headers.get('Retry-After') : null;
  if (!header) {
    return null;
  }
  const secs = Number(header);
  if (Number.isFinite(secs)) {
    return Math.max(0, secs * 1000);
  }
  const date = Date.parse(header);
  return Number.isNaN(date) ? null : Math.max(0, date - Date.now());
}

function backoffMs(attempt: number): number {
  const exp = retryConfig.baseBackoffMs * 2 ** attempt;
  return exp / 2 + Math.random() * (exp / 2);
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

async function fetchOnce(url: string, options: RequestInit, timeout: number): Promise<Response> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  try {
    return await Promise.race<Response>([
      fetch(url, options).catch(() => {
        throw new Error(url + ' failed');
      }),
      new Promise<Response>((_, reject) => {
        timer = setTimeout(() => reject(new Error(url + ' timeout')), timeout);
      }),
    ]);
  } finally {
    if (timer) {
      clearTimeout(timer);
    }
  }
}

export async function fetchWithTimeout(url: string, options: RequestInit, timeout: number = retryConfig.timeout): Promise<Response> {
  const method = ((options && options.method) || 'GET').toUpperCase();
  const retryTimeouts = IDEMPOTENT.has(method);
  for (let attempt = 0; ; attempt++) {
    const last = attempt >= retryConfig.maxAttempts - 1;
    let res: Response;
    try {
      res = await fetchOnce(url, options, timeout);
    } catch (err) {
      if (last || !retryTimeouts || !(err instanceof Error) || !err.message.endsWith(' timeout')) {
        throw err;
      }
      await sleep(backoffMs(attempt));
      continue;
    }
    if (res.status !== 429 || last) {
      return res;
    }
    const wait = retryAfterMs(res);
    await sleep(Math.min(wait === null ? backoffMs(attempt) : wait + Math.random() * 500, retryConfig.maxRetryAfterMs));
  }
}

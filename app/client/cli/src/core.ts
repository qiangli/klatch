import { createCipheriv, createDecipheriv, pbkdf2Sync, randomBytes, createPrivateKey } from 'node:crypto';
import { JSEncrypt } from 'jsencrypt';

export class Failure extends Error {
  constructor(public code: string) { super(code); }
}
export function required(input: any, key: string): string {
  if (typeof input[key] !== 'string' || !input[key]) throw new Failure('INVALID_INPUT');
  return input[key];
}
export function serverURL(server: string): string {
  const url = new URL(server);
  if (url.username || url.password || url.search || url.hash || url.pathname !== '/' ||
    (url.protocol !== 'https:' && !(url.protocol === 'http:' && ['localhost', '127.0.0.1', '[::1]'].includes(url.hostname)))) throw new Failure('INVALID_SERVER');
  return url.origin;
}
export function attributed(input: any): string {
  const name = required(input, 'assistant');
  if (!/^[\p{L}\p{N} _.-]{1,64}$/u.test(name)) throw new Failure('INVALID_ASSISTANT');
  return `[Assistant: ${name}] ${required(input, 'text')}`;
}
export function decrypt(data: string, iv: string, key: string): string {
  const cipher = createDecipheriv('aes-256-cbc', Buffer.from(key, 'hex'), Buffer.from(iv, 'hex'));
  return Buffer.concat([cipher.update(Buffer.from(data, 'base64')), cipher.final()]).toString('utf8');
}
export function encrypt(data: string, key: string): { messageIv: string; messageEncrypted: string } {
  const iv = randomBytes(16);
  const cipher = createCipheriv('aes-256-cbc', Buffer.from(key, 'hex'), iv);
  return { messageIv: iv.toString('hex'), messageEncrypted: Buffer.concat([cipher.update(data, 'utf8'), cipher.final()]).toString('base64') };
}
export function unlock(seal: any, password: string): { publicKey: string; privateKey: string } {
  try {
    const key = pbkdf2Sync(password, Buffer.from(seal.passwordSalt, 'hex'), 1024, 32, 'sha1');
    const privateKey = decrypt(seal.privateKeyEncrypted, seal.privateKeyIv, key.toString('hex'));
    createPrivateKey({key: Buffer.from(privateKey, 'base64'), format: 'der', type: 'pkcs1'});
    return { publicKey: seal.publicKey, privateKey };
  } catch { throw new Failure('SEAL_UNLOCK_FAILED'); }
}
export function channelKey(detail: any, seal?: {publicKey: string; privateKey: string}): string | undefined {
  if (detail.dataType !== 'sealed') return undefined;
  if (!seal) throw new Failure('SEAL_LOCKED');
  try {
    const entry = JSON.parse(detail.data).seals.find((s: any) => s.publicKey === seal.publicKey);
    const rsa = new JSEncrypt();
    rsa.setPrivateKey(seal.privateKey);
    const key = rsa.decrypt(entry.sealedKey);
    if (!key || !/^[a-f0-9]{64}$/i.test(key)) throw new Error();
    return key;
  } catch { throw new Failure('SEAL_UNLOCK_FAILED'); }
}
export type Session = { server: string; token: string; guid: string };
export class API {
  constructor(public session: Session, public auth = 'agent', private transport: typeof fetch = fetch) {}
  async request(path: string, method = 'GET', body?: unknown, query: Record<string, string> = {}, raw = false): Promise<any> {
    const url = new URL(path, this.session.server);
    url.search = new URLSearchParams({[this.auth]: this.session.token, ...query}).toString();
    let response: Response;
    try {
      response = await this.transport(url, { method, redirect: 'error', signal: AbortSignal.timeout(30000),
        body: body instanceof FormData || typeof body === 'string' ? body : body === undefined ? undefined : JSON.stringify(body) });
    } catch { throw new Failure('NETWORK_ERROR'); }
    if (!response.ok) throw new Failure(response.status === 401 ? 'AUTH_REQUIRED' : response.status === 403 ? 'FORBIDDEN' : response.status === 429 ? 'RATE_LIMITED' : 'HTTP_' + response.status);
    if (raw) return Buffer.from(await response.arrayBuffer());
    const text = await response.text();
    return text ? JSON.parse(text) : null;
  }
}
export const id = (value: string) => encodeURIComponent(value);

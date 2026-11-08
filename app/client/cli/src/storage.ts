import { constants } from 'node:fs';
import { open, mkdir, rename, unlink } from 'node:fs/promises';
import { dirname } from 'node:path';
import { randomUUID } from 'node:crypto';
import { readInput, INPUT_LIMIT } from './input.js';
import { Failure, serverURL, type Session } from './core.js';

export async function privateRead(path: string): Promise<string> {
  const file = await open(path, constants.O_RDONLY | constants.O_NOFOLLOW);
  try {
    const stat = await file.stat();
    if (!stat.isFile() || (process.platform !== 'win32' && ((stat.mode & 0o077) !== 0 || stat.uid !== process.getuid!()))) throw new Failure('UNSAFE_FILE');
    if (stat.size > INPUT_LIMIT) throw new Failure('INPUT_TOO_LARGE');
    return await readInput(file.createReadStream({autoClose:false}));
  } finally { await file.close(); }
}
export async function loadSession(path: string): Promise<Session> {
  const value = JSON.parse(await privateRead(path));
  if (typeof value.token !== 'string' || !value.token || typeof value.guid !== 'string') throw new Failure('INVALID_SESSION');
  return {server: serverURL(value.server), token: value.token, guid: value.guid};
}
export async function saveSession(path: string, session: Session): Promise<void> {
  await mkdir(dirname(path), { recursive: true, mode: 0o700 });
  const temporary = path + '.' + randomUUID();
  try {
    const file = await open(temporary, 'wx', 0o600);
    try { await file.writeFile(JSON.stringify(session)); await file.sync(); } finally { await file.close(); }
    await rename(temporary, path);
  } finally { await unlink(temporary).catch(() => {}); }
}

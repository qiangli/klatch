import { Failure } from './core.js';

export const INPUT_LIMIT = 1024 * 1024;
// Decode once after collecting bounded bytes: UTF-8 characters may span chunks.
export async function readInput(stream: AsyncIterable<Uint8Array>): Promise<string> {
  const chunks: Buffer[] = [];
  let size = 0;
  for await (const chunk of stream) {
    size += chunk.byteLength;
    if (size > INPUT_LIMIT) throw new Failure('INPUT_TOO_LARGE');
    chunks.push(Buffer.from(chunk));
  }
  return Buffer.concat(chunks).toString('utf8');
}

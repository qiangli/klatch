#!/usr/bin/env node
import { homedir } from 'node:os';
import { join } from 'node:path';
import { privateRead } from './storage.js';
import { Failure } from './core.js';
import { commands, execute, redact } from './commands.js';
import { watch } from './watch.js';

export function parseArgs(args:string[]) {
  const command=args.shift();
  if (!command || !commands.includes(command)) throw new Failure('USAGE');
  let input='-', session=join(homedir(),'.config','klatch','session.json');
  const seen=new Set<string>();
  while (args.length) {
    const flag=args.shift()!;
    if (seen.has(flag)) throw new Failure('USAGE'); seen.add(flag);
    if (flag === '--json') continue;
    if (!['--input','--session'].includes(flag) || !args.length) throw new Failure('USAGE');
    const value=args.shift()!;
    if (flag === '--input') input=value; else session=value;
  }
  return {command,input,session};
}
async function main() {
  if (process.argv.length === 3 && ['--help','help'].includes(process.argv[2])) {
    process.stdout.write(JSON.stringify({commands,usage:'klatch COMMAND [--input FILE|-] [--session FILE] [--json]'})+'\n'); return;
  }
  const options=parseArgs(process.argv.slice(2));
  let text='';
  if (options.input === '-') {
    if (process.stdin.isTTY) throw new Failure('JSON_INPUT_REQUIRED');
    for await (const chunk of process.stdin) { text+=chunk; if (text.length>1024*1024) throw new Failure('INPUT_TOO_LARGE'); }
  } else text=await privateRead(options.input);
  if (text.length>1024*1024) throw new Failure('INPUT_TOO_LARGE');
  const input=JSON.parse(text || '{}');
  if (!input || typeof input !== 'object' || Array.isArray(input)) throw new Failure('INVALID_INPUT');
  const emit=(result:any)=>process.stdout.write(JSON.stringify({ok:true,result:redact(result)})+'\n');
  if (options.command === 'watch') {
    const controller=new AbortController();
    const stop=()=>controller.abort();
    process.once('SIGINT',stop); process.once('SIGTERM',stop);
    try { await watch(input,options.session,emit,controller.signal); }
    finally { process.removeListener('SIGINT',stop); process.removeListener('SIGTERM',stop); }
  } else emit(await execute(options.command,input,options.session));
}
main().catch(error=>{
  const code=error instanceof Failure ? error.code : error instanceof SyntaxError ? 'INVALID_JSON' : 'OPERATION_FAILED';
  process.stderr.write(JSON.stringify({ok:false,error:{code}})+'\n');
  process.exitCode=code === 'USAGE' || code === 'INVALID_INPUT' || code === 'INVALID_JSON' ? 2 : 1;
});

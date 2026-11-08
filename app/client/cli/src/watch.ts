import WebSocket from 'ws';
import { Connection } from './sdk-connection.js';
import { loadSession } from './storage.js';
import { execute } from './commands.js';
import { Failure } from './core.js';

// The SDK logs malformed frames directly. Validate before handing frames to it,
// so a server cannot reflect credentials or terminal controls into CLI logs.
export function safeSocket(url: string): WebSocket {
  const socket = new WebSocket(url);
  Object.defineProperty(socket, 'onmessage', {set(handler: (event:any)=>void) {
    socket.on('message', bytes => {
      const data=bytes.toString();
      if (data !== '') {
        try {
          const parsed=JSON.parse(data);
          if (!parsed || typeof parsed !== 'object') throw new Error();
        } catch { socket.close(); return; }
      }
      handler({data});
    });
  }});
  return socket;
}
export async function watch(input:any, sessionPath:string, emit:(data:any)=>void, signal:AbortSignal):Promise<void> {
  const session=await loadSession(sessionPath);
  // The status endpoint authenticates app sessions, not contact tokens.
  // Remote changes notify this account's card revision (NotifyChannelRevision);
  // SDK Session also drives contact sync from that local revision stream.
  // execute() below resolves cardId to the remote scope for every snapshot.
  const url=new URL(session.server);
  // Connection uses the browser WebSocket interface, supplied by ws in Node.
  (globalThis as any).WebSocket = function(url:string) { return safeSocket(url); };
  const connection=new Connection({info(){},warn(){},error(){}},session.token,url.host + url.pathname.replace(/\/$/, ''),url.protocol === 'https:');
  let pending=false, syncing=false, stopped=false;
  let failure:unknown;
  let finish!:()=>void;
  const done=new Promise<void>(resolve=>{ finish=resolve; });
  const sync=async()=>{
    pending=true;
    if (syncing || stopped) return;
    syncing=true;
    try {
      while (pending && !stopped) {
        pending=false;
        const result=await execute(input.channelId ? 'messages-list' : 'conversations-list',input,sessionPath);
        if (!stopped) emit({event:'snapshot',result});
      }
    } catch (error) { failure=error; stopped=true; finish(); }
    finally { syncing=false; }
  };
  const stop=()=>{ stopped=true; finish(); };
  signal.addEventListener('abort',stop,{once:true});
  connection.addRevisionListener(()=>{ void sync(); });
  let connected=false;
  connection.addStatusListener(status=>{
    if (stopped) return;
    emit({event:'status',status});
    if (status === 'disconnected') connected=false;
    if (status === 'connected' && !connected) { connected=true; void sync(); }
    if (status === 'closed') { failure=new Failure('AUTH_REQUIRED'); stop(); }
  });
  if (signal.aborted) stop(); else void sync();
  await done;
  signal.removeEventListener('abort',stop);
  await connection.close();
  if (failure) throw failure;
}

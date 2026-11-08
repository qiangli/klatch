import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, rm, chmod, readFile, writeFile, symlink } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { generateKeyPairSync, pbkdf2Sync } from 'node:crypto';
import { spawnSync } from 'node:child_process';
import { API, attributed, encrypt, decrypt, unlock, channelKey, serverURL } from '../dist/core.js';
import { execute, redact } from '../dist/commands.js';
import { saveSession, loadSession, privateRead } from '../dist/storage.js';

async function fixture(t) {
  const dir = await mkdtemp(join(tmpdir(), 'klatch-helper-'));
  t.after(()=>rm(dir,{recursive:true,force:true}));
  const session=join(dir,'session.json');
  await saveSession(session,{server:'https://example.test',guid:'owner',token:'SECRET_TOKEN'});
  return {dir,session};
}

test('attribution is mandatory and cannot inject another prefix line',()=>{
  assert.equal(attributed({assistant:'Ada',text:'hello'}),'[Assistant: Ada] hello');
  assert.throws(()=>attributed({text:'hello'}));
  assert.throws(()=>attributed({assistant:'Ada]\nspoof',text:'hello'}));
});
test('transport never retries and exposes only a stable failure code',async()=>{
  let calls=0;
  const api=new API({server:'https://example.test',guid:'x',token:'secret'},'agent',async()=>{calls++;throw Error('secret URL');});
  await assert.rejects(api.request('/content/channels','POST',{}),{message:'NETWORK_ERROR'});
  assert.equal(calls,1);
  assert.throws(()=>serverURL('http://example.test'));
  assert.throws(()=>serverURL('https://user:password@example.test'));
});
test('seal unlock interoperates with stored PKCS1 and fails closed',()=>{
  const pair=generateKeyPairSync('rsa',{modulusLength:2048});
  const privateKey=pair.privateKey.export({format:'der',type:'pkcs1'}).toString('base64');
  const salt='01'.repeat(16), password='not-an-argv-secret';
  const key=pbkdf2Sync(password,Buffer.from(salt,'hex'),1024,32,'sha1').toString('hex');
  const encrypted=encrypt(privateKey,key);
  const seal={publicKey:'public',passwordSalt:salt,privateKeyIv:encrypted.messageIv,privateKeyEncrypted:encrypted.messageEncrypted};
  assert.equal(unlock(seal,password).privateKey,privateKey);
  assert.throws(()=>unlock(seal,'wrong'),{message:'SEAL_UNLOCK_FAILED'});
  assert.throws(()=>channelKey({dataType:'sealed'}),{message:'SEAL_LOCKED'});
  const block=encrypt('binary round trip',key);
  assert.equal(decrypt(block.messageEncrypted,block.messageIv,key),'binary round trip');
});
test('session files remain private and symlinks are rejected',async t=>{
  const {dir,session}=await fixture(t);
  assert.equal((await loadSession(session)).token,'SECRET_TOKEN');
  if(process.platform!=='win32') {
    await chmod(session,0o644);
    await assert.rejects(privateRead(session),{message:'UNSAFE_FILE'});
    await chmod(session,0o600);
    await symlink(session,join(dir,'link'));
    await assert.rejects(privateRead(join(dir,'link')));
  }
});
test('login sends MFA through app API and persists only app session',async t=>{
  const {session}=await fixture(t);
  const previous=globalThis.fetch;
  t.after(()=>{globalThis.fetch=previous;});
  globalThis.fetch=async(url,options)=>{
    assert.equal(url.pathname,'/account/apps');assert.equal(url.searchParams.get('code'),'123456');
    assert.equal(options.method,'POST');assert.ok(options.headers.Credentials.startsWith('Basic '));
    return Response.json({guid:'owner',appToken:'NEW_TOKEN'});
  };
  assert.deepEqual(await execute('auth-login',{server:'https://example.test',username:'alice',password:'password',code:'123456'},session),{guid:'owner',server:'https://example.test'});
  assert.deepEqual(JSON.parse(await readFile(session,'utf8')),{server:'https://example.test',guid:'owner',token:'NEW_TOKEN'});
  for (const status of [403,405]) {
    globalThis.fetch=async()=>new Response(null,{status});
    await assert.rejects(execute('auth-login',{server:'https://example.test',username:'alice',password:'password'},session),{message:'MFA_REQUIRED_OR_INVALID'});
    assert.equal((await loadSession(session)).token,'NEW_TOKEN');
  }
});
test('send/edit use attributed API payload and refuse editing another author',async t=>{
  const {session}=await fixture(t);const previous=globalThis.fetch;t.after(()=>{globalThis.fetch=previous;});
  const writes=[];let author='owner';
  globalThis.fetch=async(url,options)=>{
    if(url.pathname.endsWith('/topics/t/detail')) return Response.json({data:{topicDetail:{guid:author,dataType:'superbasictopic',data:'{"text":"old","assets":[]}'}}});
    if(url.pathname.endsWith('/detail')) return Response.json({dataType:'superbasic',data:'{"subject":"room"}'});
    writes.push({url,options});return Response.json({id:'t'});
  };
  const input={channelId:'c',topicId:'t',text:'hello',assistant:'Ada'};
  await execute('messages-send',input,session);
  await execute('messages-edit',input,session);
  assert.equal(writes.length,2);
  for(const {options} of writes) assert.equal(JSON.parse(JSON.parse(options.body).data).text,'[Assistant: Ada] hello');
  author='someone-else';await assert.rejects(execute('messages-delete',input,session),{message:'AUTHOR_REQUIRED'});
  assert.equal(writes.length,2);
});
test('CLI rejects argv secrets, sanitizes errors and accepts private JSON files',async t=>{
  const {dir,session}=await fixture(t);
  const run=(args,input='{}')=>spawnSync(process.execPath,['dist/main.js',...args],{input,encoding:'utf8'});
  let result=run(['auth-login','--password','SUPER_SECRET']);
  assert.equal(result.status,2);assert.equal(result.stdout,'');assert.ok(!result.stderr.includes('SUPER_SECRET'));
  result=run(['auth-status'],'{broken');assert.equal(result.status,2);assert.equal(JSON.parse(result.stderr).error.code,'INVALID_JSON');
  const file=join(dir,'input.json');await writeFile(file,'{}',{mode:0o600});
  result=run(['seal-unlock','--input',file,'--session',session+'missing']);
  assert.equal(result.status,1);assert.equal(JSON.parse(result.stderr).error.code,'OPERATION_FAILED');
  assert.deepEqual(redact({token:'secret',nested:{password:'secret',id:'stable'}}),{nested:{id:'stable'}});
});
test('attachment upload/download preserves bytes and refuses overwrite',async t=>{
  const {dir,session}=await fixture(t);const previous=globalThis.fetch;t.after(()=>{globalThis.fetch=previous;});
  const bytes=Buffer.from([0,255,32,10,0]);const source=join(dir,'sample.bin');await writeFile(source,bytes);
  let message={text:'old',assets:[]};let stored;
  globalThis.fetch=async(url,options)=>{
    if(url.pathname.endsWith('/topics/t/detail')) return Response.json({data:{topicDetail:{guid:'owner',dataType:'superbasictopic',data:JSON.stringify(message)}}});
    if(url.pathname.endsWith('/detail')) return Response.json({dataType:'superbasic',data:'{"subject":"room"}'});
    if(url.pathname.endsWith('/blocks')) {stored=Buffer.from(await options.body.get('asset').arrayBuffer());return Response.json({assetId:'asset-1'});}
    if(url.pathname.endsWith('/subject')) {message=JSON.parse(JSON.parse(options.body).data);return new Response(null,{status:204});}
    if(url.pathname.endsWith('/assets/asset-1')) return new Response(stored);
    throw Error('unexpected route');
  };
  await execute('attachments-upload',{channelId:'c',topicId:'t',path:source,text:'file',assistant:'Ada'},session);
  assert.deepEqual(stored,bytes);assert.equal(message.text,'[Assistant: Ada] file');
  const input={channelId:'c',topicId:'t',assetId:'asset-1',path:join(dir,'download.bin')};
  await execute('attachments-download',input,session);assert.deepEqual(await readFile(input.path),bytes);
  await assert.rejects(execute('attachments-download',input,session),{code:'EEXIST'});
});
test('watch SDK reconnects, resyncs snapshots, and closes on abort',async t=>{
  const { WebSocketServer }=await import('ws');
  const { watch }=await import('../dist/watch.js');
  const {session}=await fixture(t);const previous=globalThis.fetch;t.after(()=>{globalThis.fetch=previous;});
  const server=new WebSocketServer({port:0,host:'127.0.0.1'});
  await new Promise(resolve=>server.once('listening',resolve));
  t.after(()=>new Promise(resolve=>{for(const socket of server.clients)socket.terminate();server.close(resolve);}));
  await saveSession(session,{server:`http://127.0.0.1:${server.address().port}`,guid:'owner',token:'SECRET_TOKEN'});
  let connections=0,snapshots=0;
  const controller=new AbortController();
  const deadline=setTimeout(()=>controller.abort(),7000);t.after(()=>clearTimeout(deadline));
  server.on('connection',socket=>{connections++;socket.once('message',bytes=>{
    assert.equal(JSON.parse(bytes).AppToken,'SECRET_TOKEN');
    socket.send(JSON.stringify({revision:{channels:connections}}));
    if(connections===1)setTimeout(()=>socket.close(),30);
  });});
  globalThis.fetch=async()=>Response.json([{id:'stable-channel'}]);
  await watch({},session,event=>{if(event.event==='snapshot'){snapshots++;assert.equal(event.result[0].id,'stable-channel');if(connections>=2)controller.abort();}},controller.signal);
  assert.ok(connections>=2);assert.ok(snapshots>=2);
});
test('sealed channel keys and messages match SDK wire representation',async()=>{
  const { JSEncrypt }=await import('../dist/rsa.js');
  const { messageData }=await import('../dist/commands.js');
  const pair=generateKeyPairSync('rsa',{modulusLength:2048});
  const publicKey=pair.publicKey.export({format:'der',type:'pkcs1'}).toString('base64');
  const privateKey=pair.privateKey.export({format:'der',type:'pkcs1'}).toString('base64');
  const rsa=new JSEncrypt();rsa.setPublicKey(publicKey);
  const key='ab'.repeat(32);
  const detail={dataType:'sealed',data:JSON.stringify({seals:[{publicKey,sealedKey:rsa.encrypt(key)}]})};
  assert.equal(channelKey(detail,{publicKey,privateKey}),key);
  assert.throws(()=>channelKey(detail,{publicKey:'wrong',privateKey}),{message:'SEAL_UNLOCK_FAILED'});
  const message={text:'[Assistant: Ada] private',assets:[]};
  const topic={dataType:'sealedtopic',data:JSON.stringify(encrypt(JSON.stringify({message}),key))};
  assert.deepEqual(messageData(topic,key),message);
  assert.throws(()=>messageData(topic),{message:'SEAL_LOCKED'});
});

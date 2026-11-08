import { readFile, writeFile, unlink } from 'node:fs/promises';
import { basename, extname } from 'node:path';
import { randomBytes } from 'node:crypto';
import { JSEncrypt } from 'jsencrypt';
import { API, Failure, required, attributed, serverURL, id, unlock, channelKey, encrypt, decrypt, type Session } from './core.js';
import { loadSession, saveSession } from './storage.js';

export const commands = ['auth-login','auth-status','auth-logout','seal-unlock','contacts-list','contacts-invite','contacts-accept','conversations-list','conversations-show','conversations-create','conversations-rename','conversations-leave','members-list','members-add','members-remove','messages-list','messages-show','messages-send','messages-edit','messages-delete','attachments-upload','attachments-download','watch'];
export function redact(value: any): any {
  if (Array.isArray(value)) return value.map(redact);
  if (value && typeof value === 'object') return Object.fromEntries(Object.entries(value).filter(([k]) => !/token|password|privatekey|signature|sealedkey/i.test(k)).map(([k,v]) => [k, redact(v)]));
  return value;
}
function remoteServer(node: string, fallback: string): string {
  return node ? serverURL(node.includes('://') ? node : `https://${node}`) : fallback;
}
async function sealFor(api: API, input: any) {
  return input.sealPassword ? unlock((await api.request('/account/status')).seal, required(input,'sealPassword')) : undefined;
}
export async function scope(api: API, input: any): Promise<API> {
  if (!input.cardId) return api;
  const path = '/contact/cards/' + id(required(input,'cardId'));
  const profile = await api.request(path + '/profile');
  const detail = await api.request(path + '/detail');
  if (detail.status !== 'connected' || !detail.token) throw new Failure('CONTACT_NOT_CONNECTED');
  return new API({server: remoteServer(profile.node, api.session.server), token: `${api.session.guid}.${detail.token}`, guid: api.session.guid}, 'contact');
}
export function messageData(detail: any, key?: string): any {
  const data = JSON.parse(detail.data || '{}');
  if (detail.dataType !== 'sealedtopic') return data;
  if (!key) throw new Failure('SEAL_LOCKED');
  return JSON.parse(decrypt(data.messageEncrypted, data.messageIv, key)).message;
}
function subject(message: any, key?: string) {
  return {dataType: key ? 'sealedtopic' : 'superbasictopic', data: JSON.stringify(key ? encrypt(JSON.stringify({message}), key) : message)};
}
async function connect(api: API, cardId: string) {
  const path = '/contact/cards/' + id(cardId);
  await api.request(path + '/status', 'PUT', JSON.stringify('connecting'));
  const profile = await api.request(path + '/profile');
  const message = await api.request(path + '/openMessage');
  const remote = new API({...api.session, server: remoteServer(profile.node, api.session.server), token: ''}, '');
  const contact = await remote.request('/contact/openMessage', 'PUT', message);
  if (contact.status === 'connected') await api.request(path + '/status', 'PUT', JSON.stringify('connected'), {
    token: contact.token, viewRevision: '1', articleRevision: String(contact.articleRevision), channelRevision: String(contact.channelRevision), profileRevision: String(contact.profileRevision),
  });
  return {cardId, status: contact.status};
}
export async function execute(command: string, input: any, sessionPath: string): Promise<any> {
  if (!commands.includes(command)) throw new Failure('UNKNOWN_COMMAND');
  if (command === 'auth-login') {
    const server = serverURL(required(input,'server'));
    const auth = Buffer.from(`${required(input,'username')}:${required(input,'password')}`).toString('base64');
    const url = new URL('/account/apps',server);
    url.search = new URLSearchParams({appName:'Klatch CLI',appVersion:'0.1.0',platform:'node',deviceToken:'',pushType:'', ...(input.mfa || input.code ? {code: String(input.mfa || input.code)} : {})}).toString();
    let response: Response;
    try { response = await fetch(url,{method:'POST',headers:{Authorization:'Basic '+auth,Credentials:'Basic '+auth},body:'[]',redirect:'error',signal:AbortSignal.timeout(30000)}); }
    catch { throw new Failure('NETWORK_ERROR'); }
    if (!response.ok) throw new Failure(response.status === 401 ? 'AUTH_REQUIRED' : response.status === 403 ? 'MFA_REQUIRED_OR_INVALID' : 'HTTP_'+response.status);
    const login = await response.json() as any;
    if (!login.appToken || !login.guid) throw new Failure('INVALID_RESPONSE');
    await saveSession(sessionPath,{server,token:login.appToken,guid:login.guid});
    return {guid:login.guid,server};
  }
  const session = await loadSession(sessionPath);
  const api = new API(session);
  if (command === 'auth-status') { const status = await api.request('/account/status'); return {guid:session.guid,server:session.server,mfaEnabled:status.mfaEnabled,sealable:status.sealable,disabled:status.disabled}; }
  if (command === 'auth-logout') { await api.request('/account/apps','DELETE'); await unlink(sessionPath); return {loggedOut:true}; }
  if (command === 'seal-unlock') { const seal = unlock((await api.request('/account/status')).seal, required(input,'sealPassword')); return {unlocked:true,publicKey:seal.publicKey}; }
  if (command === 'contacts-list') return redact(await api.request('/contact/cards'));
  if (command === 'contacts-invite') {
    const remote = new API({...session,server:input.server ? serverURL(input.server) : session.server,token:''},'');
    const listing = await remote.request('/account/listing/'+id(required(input,'guid'))+'/message');
    const card = await api.request('/contact/cards','POST',listing);
    try { return await connect(api,card.id); } catch { throw new Failure('CONTACT_CONNECT_INCOMPLETE'); }
  }
  if (command === 'contacts-accept') return connect(api,required(input,'cardId'));
  const seal = await sealFor(api,input);
  const target = await scope(api,input);
  if (command === 'conversations-list') return redact(await target.request('/content/channels','GET',undefined,{types:JSON.stringify(['sealed','superbasic']),viewRevision:'1'}));
  if (command === 'conversations-create') {
    if (input.cardId) throw new Failure('OWNER_REQUIRED');
    const cards: string[] = input.cards || [];
    if (!Array.isArray(cards) || !cards.every(c=>typeof c === 'string')) throw new Failure('INVALID_INPUT');
    let data: any = {subject:required(input,'title')};
    let dataType = 'superbasic';
    if (input.sealed !== false) {
      if (!seal) throw new Failure('SEAL_LOCKED');
      const key = randomBytes(32).toString('hex');
      const publicKeys = [seal.publicKey];
      for (const card of cards) publicKeys.push(required(await api.request('/contact/cards/'+id(card)+'/profile'),'seal'));
      const seals = publicKeys.map(publicKey=>{ const rsa = new JSEncrypt(); rsa.setPublicKey(publicKey); const sealedKey=rsa.encrypt(key); if (!sealedKey) throw new Failure('INVALID_SEAL'); return {publicKey,sealedKey}; });
      const encrypted = encrypt(JSON.stringify(data),key);
      data = {subjectEncrypted:encrypted.messageEncrypted,subjectIv:encrypted.messageIv,seals}; dataType='sealed';
    }
    return api.request('/content/channels','POST',{dataType,data:JSON.stringify(data),cards,groups:[]});
  }
  const path = '/content/channels/'+id(required(input,'channelId'));
  const detail = await target.request(path+'/detail');
  const key = channelKey(detail,seal);
  const channelData = () => { const data=JSON.parse(detail.data); return key ? JSON.parse(decrypt(data.subjectEncrypted,data.subjectIv,key)) : data; };
  if (command === 'conversations-show') return {...redact(detail),channelId:input.channelId,data:channelData()};
  if (command === 'members-list') return {channelId:input.channelId,members:detail.members,cards:detail.contacts?.cards};
  if (command === 'conversations-leave') {
    if (!input.cardId) throw new Failure('CONTACT_SCOPE_REQUIRED'); // owner DELETE destroys the conversation
    await target.request(path,'DELETE'); return {left:true,channelId:input.channelId};
  }
  if (['conversations-rename','members-add','members-remove'].includes(command)) {
    if (input.cardId) throw new Failure('OWNER_REQUIRED');
    if (command === 'conversations-rename') {
      let data = {...channelData(),subject:required(input,'title')};
      if (key) { const encrypted=encrypt(JSON.stringify(data),key); data={...JSON.parse(detail.data),subjectEncrypted:encrypted.messageEncrypted,subjectIv:encrypted.messageIv}; }
      await api.request(path+'/subject','PUT',{dataType:detail.dataType,data:JSON.stringify(data)});
    } else {
      const memberId=required(input,'memberCardId');
      if (key && command === 'members-add') {
        const publicKey=required(await api.request('/contact/cards/'+id(memberId)+'/profile'),'seal');
        const rsa=new JSEncrypt(); rsa.setPublicKey(publicKey); const sealedKey=rsa.encrypt(key);
        if (!sealedKey) throw new Failure('INVALID_SEAL');
        const data=JSON.parse(detail.data); data.seals.push({publicKey,sealedKey});
        await api.request(path+'/subject','PUT',{dataType:detail.dataType,data:JSON.stringify(data)});
      }
      await api.request(path+'/cards/'+id(memberId),command === 'members-add' ? 'PUT' : 'DELETE');
    }
    return {channelId:input.channelId};
  }
  if (command === 'messages-list') {
    const query: Record<string,string> = {};
    for (const field of ['count','begin','end','revision']) if (input[field] !== undefined) {
      if (!Number.isSafeInteger(input[field]) || input[field] < 0) throw new Failure('INVALID_INPUT'); query[field]=String(input[field]);
    }
    const topics=await target.request(path+'/topics','GET',undefined,query);
    return topics.map((topic:any)=>({...topic,data:topic.data ? {...topic.data,topicDetail:{...topic.data.topicDetail,data:messageData(topic.data.topicDetail,key)}} : undefined}));
  }
  if (command === 'messages-send') return target.request(path+'/topics','POST',subject({text:attributed(input),textColor:'',textSize:'',assets:[]},key),{confirm:'true'});
  const topicPath=path+'/topics/'+id(required(input,'topicId'));
  const topicResponse=await target.request(topicPath+'/detail');
  const topic=topicResponse.data?.topicDetail;
  if (!topic) throw new Failure('INVALID_RESPONSE');
  const message=messageData(topic,key);
  if (command === 'messages-show') return {topicId:input.topicId,...topic,data:message};
  if (['messages-edit','messages-delete','attachments-upload'].includes(command) && topic.guid !== session.guid) throw new Failure('AUTHOR_REQUIRED');
  if (command === 'messages-delete') { await target.request(topicPath,'DELETE'); return {topicId:input.topicId,deleted:true}; }
  if (command === 'messages-edit') { await target.request(topicPath+'/subject','PUT',subject({...message,text:attributed(input)},key),{confirm:'true'}); return {topicId:input.topicId}; }
  if (command === 'attachments-upload') {
    const text=attributed(input);
    const filePath=required(input,'path');
    const bytes=await readFile(filePath);
    if (bytes.length > 16*1024*1024) throw new Failure('FILE_TOO_LARGE');
    const extension=extname(filePath).slice(1), label=basename(filePath,extname(filePath));
    let asset: any;
    if (key) {
      const encrypted=encrypt(bytes.toString('base64'),key);
      const result=await target.request(topicPath+'/blocks','POST',encrypted.messageEncrypted);
      asset={encrypted:{type:'binary',label,extension,parts:[{partId:result.assetId,blockIv:encrypted.messageIv}]}};
    } else {
      const form=new FormData(); form.append('asset',new Blob([bytes]),basename(filePath));
      const result=await target.request(topicPath+'/blocks','POST',form,{body:'multipart'});
      asset={binary:{label,extension,data:result.assetId}};
    }
    await target.request(topicPath+'/subject','PUT',subject({...message,text,assets:[...(message.assets || []),asset]},key),{confirm:'true'});
    return {topicId:input.topicId,asset};
  }
  if (command === 'attachments-download') {
    const assetId=required(input,'assetId');
    const asset=(message.assets || []).find((a:any)=>a.binary?.data === assetId || a.encrypted?.parts?.some((p:any)=>p.partId === assetId));
    if (!asset) throw new Failure('ASSET_NOT_FOUND');
    const chunks: Buffer[]=[];
    if (asset.encrypted) {
      if (!key) throw new Failure('SEAL_LOCKED');
      for (const part of asset.encrypted.parts) {
        const bytes=await target.request(topicPath+'/assets/'+id(part.partId),'GET',undefined,{},true);
        chunks.push(Buffer.from(decrypt(bytes.toString('utf8'),part.blockIv,key),'base64'));
      }
    } else chunks.push(await target.request(topicPath+'/assets/'+id(assetId),'GET',undefined,{},true));
    const bytes=Buffer.concat(chunks);
    await writeFile(required(input,'path'),bytes,{flag:'wx',mode:0o600});
    return {topicId:input.topicId,assetId,bytes:bytes.length};
  }
  throw new Failure('UNKNOWN_COMMAND');
}

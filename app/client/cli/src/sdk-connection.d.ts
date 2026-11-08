export class Connection {
  constructor(log: {info(m:unknown):void;warn(m:unknown):void;error(m:unknown):void}, token:string, node:string, secure:boolean);
  addRevisionListener(listener:(revision:Record<string,number>)=>void):void;
  addStatusListener(listener:(status:string)=>void):void;
  close():Promise<void>;
}

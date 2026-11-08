import { build } from 'esbuild';
import { resolve } from 'node:path';
await build({entryPoints:['../../sdk/src/connection.ts'],outfile:'dist/sdk-connection.js',bundle:true,platform:'node',format:'esm',nodePaths:[resolve('node_modules')]});
await build({entryPoints:['node_modules/jsencrypt/lib/index.js'],outfile:'dist/rsa.js',bundle:true,platform:'node',format:'esm'});

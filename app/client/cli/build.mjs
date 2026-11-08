import { build } from 'esbuild';
import { resolve } from 'node:path';
await build({entryPoints:['../../sdk/src/connection.ts'],outfile:'dist/sdk-connection.js',bundle:true,platform:'node',format:'esm',nodePaths:[resolve('node_modules')]});

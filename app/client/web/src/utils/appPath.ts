// The app is served at its own directory; hash routes do not change pathname.
export function appPath(pathname: string): string {
  return pathname.replace(/(?:index\.html)?\/?$/, '').replace(/\/$/, '')
}

export function appNode(host: string, pathname: string): string {
  return host + appPath(pathname)
}

export function accessLink(origin: string, pathname: string, mode: 'add' | 'reset', token: string): string {
  return `${origin}${appPath(pathname)}/#/access?${mode}=${encodeURIComponent(token)}`
}

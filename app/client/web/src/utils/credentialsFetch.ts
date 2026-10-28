// The satellite front removes Authorization before forwarding app requests.
export function installCredentialsFetch(): () => void {
  const originalFetch = window.fetch

  window.fetch = (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    const request = typeof Request !== 'undefined' && input instanceof Request
    const url = request ? input.url : input.toString()
    if (new URL(url, window.location.href).origin !== window.location.origin) {
      return originalFetch(input, init)
    }

    const headers = new Headers(init?.headers ?? (request ? input.headers : undefined))
    const authorization = headers.get('Authorization')
    if (!authorization || !/^Basic\s+\S+/i.test(authorization)) {
      return originalFetch(input, init)
    }

    headers.set('Credentials', authorization)
    return originalFetch(input, { ...init, headers })
  }

  return () => { window.fetch = originalFetch }
}

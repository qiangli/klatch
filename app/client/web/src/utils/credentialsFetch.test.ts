import { DatabagSDK } from 'databag-client-sdk'
import { installCredentialsFetch } from './credentialsFetch'

describe('same-origin Basic credentials', () => {
  const originalFetch = window.fetch
  let restore: () => void

  afterEach(() => {
    restore?.()
    window.fetch = originalFetch
  })

  test('copies Basic authorization from fetch options', async () => {
    const fetchMock = jest.fn().mockResolvedValue({ status: 200 })
    window.fetch = fetchMock
    restore = installCredentialsFetch()

    await fetch('/account/apps', { headers: { Authorization: 'Basic ZGVtbzpwYXNz' } })

    for (const [, init] of fetchMock.mock.calls) {
      const headers = new Headers(init.headers)
      expect(headers.get('Authorization')).toBe('Basic ZGVtbzpwYXNz')
      expect(headers.get('Credentials')).toBe('Basic ZGVtbzpwYXNz')
    }
  })

  test('leaves other origins and other auth schemes alone', async () => {
    const fetchMock = jest.fn().mockResolvedValue({ status: 200 })
    window.fetch = fetchMock
    restore = installCredentialsFetch()

    const external = `${window.location.protocol}//elsewhere.invalid/account/apps`
    const externalInit = { headers: { Authorization: 'Basic ZGVtbzpwYXNz' } }
    const bearerInit = { headers: { Authorization: 'Bearer token' } }
    await fetch(external, externalInit)
    await fetch('/account/apps', bearerInit)

    expect(fetchMock).toHaveBeenNthCalledWith(1, external, externalInit)
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/account/apps', bearerInit)
  })

  test('published SDK login passes a front that strips Authorization', async () => {
    const fetchMock = jest.fn().mockImplementation(async (_url: string, init: RequestInit) => {
      const headers = new Headers(init.headers)
      const forwarded = new Headers(headers)
      forwarded.delete('Authorization')
      return {
        status: forwarded.get('Credentials') === 'Basic ZGVtbzpwYXNz' ? 200 : 401,
        json: async () => ({ guid: 'test-guid', appToken: 'test-token', created: 1, pushSupported: false }),
      }
    })
    window.fetch = fetchMock
    restore = installCredentialsFetch()

    const sdk = new DatabagSDK({ channelTypes: ['sealed'] })
    const session = await sdk.login('demo', 'pass', `${window.location.host}/prefix`, window.location.protocol === 'https:', null, {
      pushType: '',
      deviceToken: '',
      notifications: [],
      deviceId: '0011',
      version: '0.0.1',
      appName: 'databag',
    })

    expect(session).toBeTruthy()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(fetchMock.mock.calls[0][0]).toContain('/prefix/account/apps?')
    const headers = new Headers(fetchMock.mock.calls[0][1].headers)
    expect(headers.get('Authorization')).toBe('Basic ZGVtbzpwYXNz')
    expect(headers.get('Credentials')).toBe('Basic ZGVtbzpwYXNz')
  })
})

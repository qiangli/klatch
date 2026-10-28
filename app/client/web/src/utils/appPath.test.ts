import { accessLink, appNode } from './appPath'

describe('app mount path', () => {
  test('keeps the prefix in API nodes', () => {
    expect(appNode('sat.example', '/matrix/h/test/app/klatch/')).toBe('sat.example/matrix/h/test/app/klatch')
    expect(appNode('sat.example', '/')).toBe('sat.example')
    expect(appNode('sat.example', '/matrix/h/test/app/klatch/index.html')).toBe('sat.example/matrix/h/test/app/klatch')
  })
  test('keeps the prefix in invite links', () => {
    expect(accessLink('https://sat.example', '/matrix/h/test/app/klatch/', 'add', 'a+b'))
      .toBe('https://sat.example/matrix/h/test/app/klatch/#/access?add=a%2Bb')
  })
})

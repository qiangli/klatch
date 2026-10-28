import { findMentionMember, getChannelMentionMembers, getMentionQuery, hasMention, insertMention, matchMentionMembers, parseMentions } from './mentions'

const members = [
  { handle: 'alice', name: 'Alice Adams' },
  { handle: 'bob_2', name: 'Robert' },
  { handle: 'carol-dev', name: 'Carol' },
]

describe('mentions', () => {
  test('parses mentions next to whitespace and punctuation', () => {
    expect(parseMentions('Hi @alice, (@bob_2)!')).toEqual([
      { handle: 'alice', start: 3, end: 9 },
      { handle: 'bob_2', start: 12, end: 18 },
    ])
  })

  test('does not parse email addresses or embedded tokens', () => {
    expect(parseMentions('mail person@example.test or word@alice and @@bob_2')).toEqual([])
  })

  test('matches members by handle or display name without changing order', () => {
    expect(matchMentionMembers(members, 'ali')).toEqual([members[0]])
    expect(matchMentionMembers(members, 'ROB')).toEqual([members[1]])
    expect(matchMentionMembers(members, '')).toEqual(members)
  })

  test('distinguishes known, unknown, and signed-in user mentions', () => {
    const [known, unknown] = parseMentions('@alice and @nobody')
    expect(findMentionMember(known, members)).toEqual(members[0])
    expect(findMentionMember(unknown, members)).toBeUndefined()
    expect(hasMention('@ALICE, hello', 'alice')).toBe(true)
    expect(hasMention('@ALICE, hello', 'alice', members)).toBe(true)
    expect(hasMention('@ALICE, hello', 'alice', members.slice(1))).toBe(false)
    expect(hasMention('person@alice.test', 'alice')).toBe(false)
    expect(hasMention(null, 'alice')).toBe(false)
  })

  test('builds owner-view candidates from all cards and excludes the owner', () => {
    const owner = { guid: 'owner-guid', handle: 'alice', name: 'Alice Adams' }
    const cards = [
      { guid: 'bob-guid', handle: 'bob_2', name: 'Robert' },
      { guid: 'carol-guid', handle: 'carol-dev', name: 'Carol' },
      { guid: 'owner-copy', handle: 'ALICE', name: 'Duplicate owner' },
    ]

    expect(getChannelMentionMembers(owner, null, ['bob-guid'], cards)).toEqual(members.slice(1))
  })

  test('builds member-view candidates from the channel owner and other members', () => {
    const viewer = { guid: 'bob-guid', handle: 'bob_2', name: 'Robert' }
    const owner = { guid: 'owner-guid', handle: 'alice', name: 'Alice Adams' }
    const cards = [
      owner,
      viewer,
      { guid: 'carol-guid', handle: 'carol-dev', name: 'Carol' },
      { guid: 'outsider-guid', handle: 'dave', name: 'Dave' },
    ]

    expect(getChannelMentionMembers(viewer, owner, ['bob-guid', 'carol-guid'], cards)).toEqual([members[0], members[2]])
  })

  test('finds an active query and inserts a complete token', () => {
    const query = getMentionQuery('hello @al', 9)
    expect(query).toEqual({ query: 'al', start: 6, end: 9 })
    expect(insertMention('hello @al!', query!, 'alice')).toEqual({ text: 'hello @alice !', caret: 13 })
    expect(getMentionQuery('person@example', 14)).toBeNull()
  })
})

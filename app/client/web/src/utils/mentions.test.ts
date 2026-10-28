import { findMentionMember, getMentionQuery, hasMention, insertMention, matchMentionMembers, parseMentions } from './mentions'

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
    expect(hasMention('person@alice.test', 'alice')).toBe(false)
    expect(hasMention(null, 'alice')).toBe(false)
  })

  test('finds an active query and inserts a complete token', () => {
    const query = getMentionQuery('hello @al', 9)
    expect(query).toEqual({ query: 'al', start: 6, end: 9 })
    expect(insertMention('hello @al!', query!, 'alice')).toEqual({ text: 'hello @alice !', caret: 13 })
    expect(getMentionQuery('person@example', 14)).toBeNull()
  })
})

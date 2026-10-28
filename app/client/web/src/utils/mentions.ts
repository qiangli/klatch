export type MentionMember = {
  handle: string
  name?: string | null
}

export type Mention = {
  handle: string
  start: number
  end: number
}

export type MentionQuery = {
  query: string
  start: number
  end: number
}

const HANDLE = '[A-Za-z0-9](?:[A-Za-z0-9_-]*[A-Za-z0-9])?'
const MENTION_PATTERN = new RegExp(`(^|[^A-Za-z0-9_@.-])@(${HANDLE})`, 'g')

export function parseMentions(text: string): Mention[] {
  const mentions: Mention[] = []
  let match: RegExpExecArray | null

  while ((match = MENTION_PATTERN.exec(text)) !== null) {
    const start = match.index + match[1].length
    mentions.push({ handle: match[2], start, end: start + match[2].length + 1 })
  }

  return mentions
}

export function findMentionMember(mention: Mention, members: MentionMember[]): MentionMember | undefined {
  const handle = mention.handle.toLocaleLowerCase()
  return members.find((member) => member.handle.toLocaleLowerCase() === handle)
}

export function hasMention(text?: string | null, handle?: string | null): boolean {
  if (!text || !handle) {
    return false
  }
  const expected = handle.toLocaleLowerCase()
  return parseMentions(text).some((mention) => mention.handle.toLocaleLowerCase() === expected)
}

export function getMentionQuery(text: string, caret: number): MentionQuery | null {
  const beforeCaret = text.slice(0, caret)
  const match = beforeCaret.match(/(?:^|[^A-Za-z0-9_@.-])@([A-Za-z0-9_-]*)$/)
  if (!match) {
    return null
  }
  const start = caret - match[1].length - 1
  return { query: match[1], start, end: caret }
}

export function matchMentionMembers(members: MentionMember[], query: string): MentionMember[] {
  const value = query.toLocaleLowerCase()
  return members.filter((member) => member.handle.toLocaleLowerCase().includes(value) || Boolean(member.name?.toLocaleLowerCase().includes(value)))
}

export function insertMention(text: string, query: MentionQuery, handle: string): { text: string; caret: number } {
  const token = `@${handle} `
  return {
    text: text.slice(0, query.start) + token + text.slice(query.end),
    caret: query.start + token.length,
  }
}

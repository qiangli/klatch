import React, { act, useRef, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { insertMention } from '../utils/mentions'
import { usePendingCaret } from './usePendingCaret.hook'

;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT = true

function MentionComposer() {
  const [value, setValue] = useState('@')
  const input = useRef<HTMLTextAreaElement>(null)
  const restoreCaret = usePendingCaret(input, value)

  const selectMention = () => {
    const inserted = insertMention(value, { query: '', start: 0, end: 1 }, 'alice')
    restoreCaret(inserted.caret)
    setValue(inserted.text)
  }

  return <textarea aria-label="Message" ref={input} value={value} onChange={(event) => setValue(event.currentTarget.value)} onKeyDown={(event) => event.key === 'Enter' && selectMention()} />
}

function typeCharacter(input: HTMLTextAreaElement, character: string) {
  const start = input.selectionStart
  const end = input.selectionEnd
  const value = `${input.value.slice(0, start)}${character}${input.value.slice(end)}`

  Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value')?.set?.call(input, value)
  input.setSelectionRange(start + 1, start + 1)
  input.dispatchEvent(new Event('input', { bubbles: true }))
}

describe('usePendingCaret', () => {
  test('restores the mention caret once and leaves later typing at the current caret', () => {
    const container = document.createElement('div')
    const root = createRoot(container)

    act(() => root.render(<MentionComposer />))
    const input = container.querySelector('textarea') as HTMLTextAreaElement

    act(() => input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true })))
    expect(input.value).toBe('@alice ')
    expect(input.selectionStart).toBe(7)

    for (const character of 'hi mvt0ind5') {
      act(() => typeCharacter(input, character))
    }

    expect(input.value).toBe('@alice hi mvt0ind5')
    expect(input.selectionStart).toBe(input.value.length)

    act(() => root.unmount())
  })
})

import { RefObject, useLayoutEffect, useRef } from 'react'

export function usePendingCaret(input: RefObject<HTMLTextAreaElement>, value: string) {
  const pendingCaret = useRef(null as number | null)

  useLayoutEffect(() => {
    const caret = pendingCaret.current
    if (caret === null) {
      return
    }

    pendingCaret.current = null
    input.current?.focus()
    input.current?.setSelectionRange(caret, caret)
  }, [input, value])

  return (caret: number) => {
    pendingCaret.current = caret
  }
}

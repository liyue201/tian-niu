import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import type { TextMessagePartProps } from '@assistant-ui/react'

export default function MarkdownText({ text }: TextMessagePartProps) {
  return (
    <ReactMarkdown remarkPlugins={[remarkGfm]}>
      {text}
    </ReactMarkdown>
  )
}
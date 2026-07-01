import { MessageSquarePlus, Pencil, Trash2, LogOut, ChevronUp, ChevronDown, User } from 'lucide-react'
import { useEffect, useState } from 'react'
import logo from '../../assets/tn.png'
import { getCurrentUser, clearAuthToken, clearCurrentUser } from '../../api'
import {
  useAui,
  ThreadListItemPrimitive,
  ThreadListPrimitive,
  useAuiState,
} from '@assistant-ui/react'

import { Button } from '../ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '../ui/dialog'
import { Input } from '../ui/input'

export default function AssistantThreadList() {
  const mainThreadId = useAuiState((s) => s.threads.mainThreadId)

  return (
    <aside
      style={{
        width: 260,
        background: 'var(--sidebar-bg)',
        borderRight: '1px solid var(--border)',
        display: 'flex',
        flexDirection: 'column',
        flexShrink: 0,
      }}
    >
      <div
        style={{
          padding: '16px 12px',
          borderBottom: '1px solid var(--border)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <img 
            src={logo} 
            alt="TianNiu AI"
            style={{ width: 24, height: 24, borderRadius: 6, objectFit: 'cover' }} 
          />
          <span style={{ fontSize: 15, fontWeight: 600, color: 'var(--text)' }}>
            天牛 AI
          </span>
        </div>
        <ThreadListPrimitive.New
          title="New Chat"
          style={{
            background: 'none',
            border: 'none',
            cursor: 'pointer',
            color: 'var(--text-muted)',
            padding: 4,
            borderRadius: 6,
            display: 'flex',
            alignItems: 'center',
          }}
        >
          <MessageSquarePlus size={18} />
        </ThreadListPrimitive.New>
      </div>

      <ThreadListPrimitive.Root
        style={{ flex: 1, overflowY: 'auto', padding: '8px 6px' }}
      >
        <ThreadListPrimitive.Items>
          {({ threadListItem }) => {
            const active = threadListItem.id === mainThreadId
            return (
              <ThreadListItemPrimitive.Root
                key={threadListItem.id}
                style={{
                  marginBottom: 8,
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                }}
              >
                <ThreadListItemPrimitive.Trigger
                  style={{
                    flex: 1,
                    minWidth: 0,
                    textAlign: 'left',
                    background: active ? 'rgba(124,58,237,0.15)' : 'transparent',
                    border: active
                      ? '1px solid rgba(124,58,237,0.4)'
                      : '1px solid transparent',
                    borderRadius: 8,
                    padding: '10px 12px',
                    cursor: 'pointer',
                    color: active ? 'var(--accent-light)' : 'var(--text)',
                    fontSize: 13,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  <ThreadListItemPrimitive.Title />
                </ThreadListItemPrimitive.Trigger>
                {threadListItem.remoteId ? (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                    <RenameThreadButton currentTitle={threadListItem.title || 'Untitled'} />
                    <DeleteThreadButton />
                  </div>
                ) : null}
              </ThreadListItemPrimitive.Root>
            )
          }}
        </ThreadListPrimitive.Items>
      </ThreadListPrimitive.Root>

      {/* User info and logout menu */}
      <div
        style={{
          padding: '12px',
          borderTop: '1px solid var(--border)',
          background: 'var(--sidebar-bg)',
        }}
      >
        <UserMenu />
      </div>
    </aside>
  )
}

function UserMenu() {
  const [open, setOpen] = useState(false)
  const user = getCurrentUser()

  const handleLogout = () => {
    clearAuthToken()
    clearCurrentUser()
    setOpen(false)
    window.location.reload()
  }

  if (!user) return null

  return (
    <div style={{ position: 'relative' }}>
      <button
        onClick={() => setOpen(!open)}
        style={{
          width: '100%',
          display: 'flex',
          alignItems: 'center',
          gap: 10,
          padding: '8px 10px',
          background: 'transparent',
          border: 'none',
          borderRadius: 8,
          cursor: 'pointer',
          color: 'var(--text)',
          textAlign: 'left',
        }}
      >
        <div
          style={{
            width: 32,
            height: 32,
            borderRadius: '50%',
            background: 'var(--accent)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: '#fff',
            fontSize: 14,
          }}
        >
          <User size={16} />
        </div>
        <div style={{ flex: 1, overflow: 'hidden' }}>
          <div
            style={{
              fontSize: 13,
              fontWeight: 500,
              color: 'var(--text)',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {user.username}
          </div>
          <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>
            {user.email || 'User'}
          </div>
        </div>
        {open ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
      </button>

      {open && (
        <div
          style={{
            position: 'absolute',
            bottom: '100%',
            left: 8,
            right: 8,
            marginBottom: 4,
            background: 'var(--sidebar-bg)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            boxShadow: '0 2px 10px rgba(0,0,0,0.1)',
            overflow: 'hidden',
          }}
        >
          <button
            onClick={handleLogout}
            style={{
              width: '100%',
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '10px 12px',
              background: 'transparent',
              border: 'none',
              cursor: 'pointer',
              color: 'var(--text)',
              textAlign: 'left',
              fontSize: 13,
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.background = 'var(--panel-bg)'
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.background = 'transparent'
            }}
          >
            <LogOut size={14} />
            <span>Logout</span>
          </button>
        </div>
      )}
    </div>
  )
}

function RenameThreadButton({ currentTitle }: { currentTitle: string }) {
  const aui = useAui()
  const [open, setOpen] = useState(false)
  const [title, setTitle] = useState(currentTitle)
  const normalizedTitle = title.trim()
  const canSave = normalizedTitle.length > 0 && normalizedTitle !== currentTitle

  useEffect(() => {
    if (!open) {
      setTitle(currentTitle)
    }
  }, [currentTitle, open])

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    event.stopPropagation()

    if (!canSave) {
      setOpen(false)
      return
    }

    aui.threadListItem().rename(normalizedTitle)
    setOpen(false)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-8 w-8 rounded-md"
          title="Rename Conversation"
          onClick={(event) => {
            event.stopPropagation()
          }}
        >
          <Pencil size={14} />
        </Button>
      </DialogTrigger>
      <DialogContent
        className="max-w-[38rem] border-0 bg-transparent p-4 shadow-none"
        style={{ padding: 20 }}
        onClick={(event) => event.stopPropagation()}
      >
        <form
          onSubmit={handleSubmit}
          className="space-y-0 overflow-hidden rounded-[28px] border border-gray-200 bg-gray-50 shadow-lg"
          style={{ borderRadius: 28, overflow: 'hidden' }}
        >
          <div
            className="flex items-start justify-between gap-4 border-b border-gray-200"
            style={{ padding: '28px 32px' }}
          >
            <DialogHeader className="gap-2">
              <DialogTitle>Rename Conversation</DialogTitle>
              <DialogDescription>
                Update the thread title shown in the sidebar.
              </DialogDescription>
            </DialogHeader>
            <span className="rounded-full border border-gray-300 bg-gray-200 px-2.5 py-1 text-[11px] font-medium uppercase tracking-[0.16em] text-gray-600">
              Thread
            </span>
          </div>
          <div style={{ padding: '28px 32px' }}>
            <div className="space-y-4">
              <label className="text-xs font-medium uppercase tracking-[0.16em] text-[var(--text-muted)]">
                Title
              </label>
              <Input
                autoFocus
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                onClick={(event) => event.stopPropagation()}
                placeholder="Conversation title"
                className="h-14 rounded-2xl border-gray-300 bg-white px-5 text-base focus-visible:ring-[var(--accent)]"
              />
              <p className="text-sm leading-6 text-[var(--text-muted)]">
                Keep it short and recognizable so it is easy to scan in the sidebar.
              </p>
            </div>
          </div>
          <div
            className="border-t border-gray-200 bg-gray-100"
            style={{ padding: '20px 32px' }}
          >
            <div className="flex items-center justify-between gap-4">
              <p className="text-xs uppercase tracking-[0.14em] text-[var(--text-muted)]">
                Enter to save
              </p>
              <div className="flex items-center gap-3">
                <Button type="button" variant="outline" className="min-w-24" onClick={() => setOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit" className="min-w-24" disabled={!canSave}>
                  Save
                </Button>
              </div>
            </div>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function DeleteThreadButton() {
  const aui = useAui()
  const [open, setOpen] = useState(false)

  const handleDelete = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    event.stopPropagation()
    aui.threadListItem().delete()
    setOpen(false)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-8 w-8 rounded-md hover:bg-red-500/10 hover:text-red-300"
          title="Delete Conversation"
          onClick={(event) => {
            event.stopPropagation()
          }}
        >
          <Trash2 size={14} />
        </Button>
      </DialogTrigger>
      <DialogContent
        className="max-w-[38rem] border-0 bg-transparent p-4 shadow-none"
        style={{ padding: 20 }}
        onClick={(event) => event.stopPropagation()}
      >
        <form
          onSubmit={handleDelete}
          className="space-y-0 overflow-hidden rounded-[28px] border border-gray-200 bg-gray-50 shadow-lg"
          style={{ borderRadius: 28, overflow: 'hidden' }}
        >
          <div
            className="flex items-start justify-between gap-4 border-b border-gray-200"
            style={{ padding: '28px 32px' }}
          >
            <DialogHeader className="gap-2">
              <DialogTitle>Delete Conversation</DialogTitle>
              <DialogDescription>
                This will permanently remove the conversation and its saved messages.
              </DialogDescription>
            </DialogHeader>
            <span className="rounded-full border border-red-200 bg-red-100 px-2.5 py-1 text-[11px] font-medium uppercase tracking-[0.16em] text-red-600">
              Danger
            </span>
          </div>
          <div style={{ padding: '28px 32px' }}>
            <p className="text-sm leading-7 text-[var(--text)]">
              This action cannot be undone. If you still need this thread, rename it instead of deleting it.
            </p>
          </div>
          <div
            className="border-t border-gray-200 bg-gray-100"
            style={{ padding: '20px 32px' }}
          >
            <div className="flex items-center justify-between gap-4">
              <p className="text-xs uppercase tracking-[0.14em] text-[var(--text-muted)]">
                Permanent action
              </p>
              <div className="flex items-center gap-3">
                <Button type="button" variant="outline" className="min-w-24" onClick={() => setOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit" className="min-w-24 bg-red-500 hover:bg-red-400">
                  Delete
                </Button>
              </div>
            </div>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
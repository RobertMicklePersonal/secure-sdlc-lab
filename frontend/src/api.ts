// Limits mirror the Go API (backend/internal/api/handler.go) so the UI can
// reject bad input before a round trip. The server still enforces them.
export const MAX_TITLE = 200
export const MAX_BODY = 5000

export interface Note {
  id: string
  title: string
  body: string
  createdAt: string
  updatedAt: string
}

export interface NoteInput {
  title: string
  body: string
}

export class ApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

const ID_PATTERN = /^[0-9a-f]{32}$/

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: { Accept: 'application/json', ...init?.headers },
    // Same-origin API: never send cookies to anywhere else.
    credentials: 'same-origin',
  })
  if (!res.ok) {
    let message = `Request failed (${res.status})`
    try {
      const data: unknown = await res.json()
      if (data && typeof data === 'object' && 'error' in data && typeof data.error === 'string') {
        message = data.error
      }
    } catch {
      // Non-JSON error body: keep the generic message.
    }
    throw new ApiError(res.status, message)
  }
  if (res.status === 204) {
    return undefined as T
  }
  return (await res.json()) as T
}

function noteUrl(id: string): string {
  // IDs come from the server, but check them anyway so a bad value can never
  // turn into a different path.
  if (!ID_PATTERN.test(id)) {
    throw new ApiError(400, 'invalid note id')
  }
  return `/api/notes/${id}`
}

export function validateNote(input: NoteInput): string | null {
  const title = input.title.trim()
  if (title === '') return 'Title is required.'
  if ([...title].length > MAX_TITLE) return `Title must be at most ${MAX_TITLE} characters.`
  if ([...input.body].length > MAX_BODY) return `Body must be at most ${MAX_BODY} characters.`
  return null
}

export function listNotes(): Promise<Note[]> {
  return request<Note[]>('/api/notes')
}

export function createNote(input: NoteInput): Promise<Note> {
  return request<Note>('/api/notes', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title: input.title.trim(), body: input.body }),
  })
}

export async function deleteNote(id: string): Promise<void> {
  await request<void>(noteUrl(id), { method: 'DELETE' })
}

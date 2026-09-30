import { useEffect, useState, type FormEvent } from 'react'
import {
  MAX_BODY,
  MAX_TITLE,
  createNote,
  deleteNote,
  listNotes,
  validateNote,
  type Note,
} from './api'

function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : 'Something went wrong.'
}

export default function App() {
  const [notes, setNotes] = useState<Note[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    // Ignore a response that lands after unmount (or StrictMode's re-run).
    let active = true
    listNotes()
      .then((list) => {
        if (active) setNotes(list)
      })
      .catch((err: unknown) => {
        if (active) setError(errorMessage(err))
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [])

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const input = { title, body }
    const invalid = validateNote(input)
    if (invalid) {
      setError(invalid)
      return
    }
    setSaving(true)
    try {
      const note = await createNote(input)
      setNotes((prev) => [note, ...prev])
      setTitle('')
      setBody('')
      setError(null)
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(id: string) {
    try {
      await deleteNote(id)
      setNotes((prev) => prev.filter((n) => n.id !== id))
      setError(null)
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  return (
    <main className="app">
      <h1>Notes</h1>

      <form className="note-form" onSubmit={handleSubmit} noValidate>
        <label>
          Title
          <input
            name="title"
            value={title}
            maxLength={MAX_TITLE}
            required
            onChange={(e) => setTitle(e.target.value)}
          />
        </label>
        <label>
          Body
          <textarea
            name="body"
            value={body}
            maxLength={MAX_BODY}
            rows={4}
            onChange={(e) => setBody(e.target.value)}
          />
        </label>
        <button type="submit" disabled={saving}>
          {saving ? 'Saving…' : 'Add note'}
        </button>
      </form>

      {error && (
        <p role="alert" className="error">
          {error}
        </p>
      )}

      {loading ? (
        <p>Loading…</p>
      ) : notes.length === 0 ? (
        <p className="empty">No notes yet.</p>
      ) : (
        <ul className="notes" aria-label="Notes">
          {notes.map((note) => (
            <li key={note.id} className="note">
              <div>
                <h2>{note.title}</h2>
                {note.body && <p>{note.body}</p>}
                <time dateTime={note.createdAt}>{new Date(note.createdAt).toLocaleString()}</time>
              </div>
              <button
                type="button"
                aria-label={`Delete ${note.title}`}
                onClick={() => void handleDelete(note.id)}
              >
                Delete
              </button>
            </li>
          ))}
        </ul>
      )}
    </main>
  )
}

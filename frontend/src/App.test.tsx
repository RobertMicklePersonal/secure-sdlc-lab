import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App'
import type { Note } from './api'

function note(id: string, title: string, body = ''): Note {
  const now = '2026-09-30T12:00:00Z'
  return { id: id.padStart(32, '0'), title, body, createdAt: now, updatedAt: now }
}

// A tiny in-memory fake of the Go API.
function fakeServer(initial: Note[]) {
  let notes = [...initial]
  let next = initial.length + 1
  return vi.fn(async (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    if (url === '/api/notes' && method === 'GET') {
      return Response.json(notes)
    }
    if (url === '/api/notes' && method === 'POST') {
      const input = JSON.parse(String(init?.body))
      const created = note(String(next++), input.title, input.body)
      notes = [created, ...notes]
      return Response.json(created, { status: 201 })
    }
    if (url.startsWith('/api/notes/') && method === 'DELETE') {
      notes = notes.filter((n) => `/api/notes/${n.id}` !== url)
      return new Response(null, { status: 204 })
    }
    return Response.json({ error: 'not found' }, { status: 404 })
  })
}

describe('App', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', fakeServer([note('1', 'Existing note', 'hello')]))
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('lists notes from the API', async () => {
    render(<App />)
    expect(await screen.findByText('Existing note')).toBeInTheDocument()
  })

  it('creates a note', async () => {
    const user = userEvent.setup()
    render(<App />)
    await screen.findByText('Existing note')

    await user.type(screen.getByLabelText('Title'), 'Buy milk')
    await user.type(screen.getByLabelText('Body'), '2 litres')
    await user.click(screen.getByRole('button', { name: 'Add note' }))

    const list = screen.getByRole('list', { name: 'Notes' })
    expect(await within(list).findByText('Buy milk')).toBeInTheDocument()
    expect(screen.getByLabelText('Title')).toHaveValue('')
  })

  it('deletes a note', async () => {
    const user = userEvent.setup()
    render(<App />)
    await screen.findByText('Existing note')

    await user.click(screen.getByRole('button', { name: 'Delete Existing note' }))

    await waitFor(() => expect(screen.queryByText('Existing note')).not.toBeInTheDocument())
    expect(screen.getByText('No notes yet.')).toBeInTheDocument()
  })

  it('blocks an empty title without calling the API', async () => {
    const user = userEvent.setup()
    render(<App />)
    await screen.findByText('Existing note')
    const fetchMock = vi.mocked(fetch)
    fetchMock.mockClear()

    await user.click(screen.getByRole('button', { name: 'Add note' }))

    expect(screen.getByRole('alert')).toHaveTextContent('Title is required.')
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('renders note text as text, not HTML', async () => {
    vi.stubGlobal('fetch', fakeServer([note('2', '<img src=x onerror=alert(1)>')]))
    const { container } = render(<App />)

    expect(await screen.findByText('<img src=x onerror=alert(1)>')).toBeInTheDocument()
    expect(container.querySelector('img')).toBeNull()
  })

  it('shows an error when the API is down', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))
    render(<App />)

    expect(await screen.findByRole('alert')).toHaveTextContent('Failed to fetch')
  })
})

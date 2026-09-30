import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, MAX_BODY, MAX_TITLE, createNote, deleteNote, listNotes, validateNote } from './api'

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('validateNote', () => {
  it('requires a non-blank title', () => {
    expect(validateNote({ title: '   ', body: '' })).toMatch(/required/)
  })

  it('counts characters, not UTF-16 units', () => {
    expect(validateNote({ title: '😀'.repeat(MAX_TITLE), body: '' })).toBeNull()
    expect(validateNote({ title: 'a'.repeat(MAX_TITLE + 1), body: '' })).toMatch(/Title/)
  })

  it('bounds the body', () => {
    expect(validateNote({ title: 'a', body: 'b'.repeat(MAX_BODY + 1) })).toMatch(/Body/)
  })
})

describe('api client', () => {
  it('posts JSON with a trimmed title', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(201, { id: 'x' }))
    vi.stubGlobal('fetch', fetchMock)

    await createNote({ title: ' hi ', body: 'there' })

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/notes')
    expect(init.method).toBe('POST')
    expect(init.credentials).toBe('same-origin')
    expect(init.headers['Content-Type']).toBe('application/json')
    expect(JSON.parse(init.body)).toEqual({ title: 'hi', body: 'there' })
  })

  it('surfaces the server error message', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse(422, { error: 'title is too long' })))

    await expect(listNotes()).rejects.toEqual(new ApiError(422, 'title is too long'))
  })

  it('falls back to a generic message for non-JSON errors', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('<html>', { status: 502 })))

    await expect(listNotes()).rejects.toThrow('Request failed (502)')
  })

  it('refuses malformed IDs instead of building a path from them', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    await expect(deleteNote('../healthz')).rejects.toThrow('invalid note id')
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('handles 204 on delete', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 204 })))

    await expect(deleteNote('0'.repeat(32))).resolves.toBeUndefined()
  })
})

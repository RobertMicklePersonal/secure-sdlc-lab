import { expect, test } from '@playwright/test'

// Checks the deployed stack from the outside: what a browser or attacker
// actually gets back, not what the config files say.

test('the UI is served with strict security headers', async ({ request }) => {
  const res = await request.get('/')
  expect(res.status()).toBe(200)
  const h = res.headers()
  expect(h['content-security-policy']).toContain("default-src 'self'")
  expect(h['content-security-policy']).toContain("script-src 'self'")
  expect(h['content-security-policy']).toContain("frame-ancestors 'none'")
  expect(h['x-content-type-options']).toBe('nosniff')
  expect(h['x-frame-options']).toBe('DENY')
  expect(h['referrer-policy']).toBe('no-referrer')
  // server_tokens off: no version number in the Server header.
  expect(h['server'] ?? '').not.toMatch(/\d/)
})

test('the CSP blocks injected inline script', async ({ page }) => {
  const violations: string[] = []
  page.on('console', (m) => {
    if (m.type() === 'error' && /Content Security Policy/i.test(m.text())) violations.push(m.text())
  })
  await page.goto('/')
  const ran = await page.evaluate(() => {
    const s = document.createElement('script')
    s.textContent = 'window.__injected = true'
    document.body.appendChild(s)
    return (window as unknown as { __injected?: boolean }).__injected === true
  })
  expect(ran).toBe(false)
  await expect.poll(() => violations.length).toBeGreaterThan(0)
})

test.describe('API input handling through the proxy', () => {
  test('rejects a non-JSON content type', async ({ request }) => {
    const res = await request.post('/api/notes', {
      headers: { 'Content-Type': 'text/plain' },
      data: 'title=hi',
    })
    expect(res.status()).toBe(415)
  })

  test('rejects unknown fields', async ({ request }) => {
    const res = await request.post('/api/notes', {
      data: { title: 'hi', admin: true },
    })
    expect(res.status()).toBe(400)
  })

  test('rejects an oversized body', async ({ request }) => {
    const res = await request.post('/api/notes', {
      headers: { 'Content-Type': 'application/json' },
      data: JSON.stringify({ title: 'big', body: 'x'.repeat(100_000) }),
    })
    expect(res.status()).toBe(413)
  })

  test('returns 404 for a malformed note id without echoing it', async ({ request }) => {
    const res = await request.get('/api/notes/%3Cscript%3E')
    expect(res.status()).toBe(404)
    expect(await res.json()).toEqual({ error: 'note not found' })
  })

  test('does not follow encoded path traversal out of the web root', async ({ request }) => {
    // nginx decodes and normalises the path before routing, so this lands on
    // the SPA fallback (index.html) instead of the API or the filesystem.
    const res = await request.get('/api/notes/..%2F..%2F..%2Fetc%2Fpasswd')
    expect(res.status()).toBeLessThan(500)
    expect(await res.text()).not.toContain('root:')
  })

  test('does not allow cross-origin reads', async ({ request }) => {
    const res = await request.get('/api/notes', { headers: { Origin: 'https://evil.example' } })
    expect(res.headers()['access-control-allow-origin']).toBeUndefined()
  })
})

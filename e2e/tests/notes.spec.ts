import { expect, test, type Page } from '@playwright/test'

// Tests share one backend and run in parallel, so each works only with notes
// it created, found by a unique title.
function uniqueTitle(label: string): string {
  return `${label} ${Date.now()}-${Math.random().toString(16).slice(2, 8)}`
}

async function addNote(page: Page, title: string, body = '') {
  await page.getByLabel('Title').fill(title)
  await page.getByLabel('Body').fill(body)
  await page.getByRole('button', { name: 'Add note' }).click()
}

function noteItem(page: Page, title: string) {
  return page.getByRole('list', { name: 'Notes' }).getByRole('listitem').filter({ hasText: title })
}

test.beforeEach(async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Notes', level: 1 })).toBeVisible()
})

test('creates a note that survives a reload', async ({ page }) => {
  const title = uniqueTitle('Groceries')
  await addNote(page, title, 'Milk and eggs')

  const item = noteItem(page, title)
  await expect(item).toBeVisible()
  await expect(item).toContainText('Milk and eggs')
  await expect(page.getByLabel('Title')).toHaveValue('')

  await page.reload()
  await expect(noteItem(page, title)).toBeVisible()
})

test('deletes a note', async ({ page }) => {
  const title = uniqueTitle('Temporary')
  await addNote(page, title)
  await expect(noteItem(page, title)).toBeVisible()

  await page.getByRole('button', { name: `Delete ${title}` }).click()
  await expect(noteItem(page, title)).toHaveCount(0)

  await page.reload()
  await expect(page.getByRole('heading', { name: title })).toHaveCount(0)
})

test('rejects a note without a title', async ({ page }) => {
  await page.getByRole('button', { name: 'Add note' }).click()
  await expect(page.getByRole('alert')).toHaveText('Title is required.')
})

test('renders markup in notes as text, never as HTML', async ({ page }) => {
  let dialogs = 0
  page.on('dialog', (d) => {
    dialogs++
    void d.dismiss()
  })
  const payload = `<img src=x onerror=alert(1)> ${uniqueTitle('xss')}`
  await addNote(page, payload, '<script>alert(2)</script>')

  const item = noteItem(page, payload)
  await expect(item).toBeVisible()
  await expect(item.getByRole('heading')).toHaveText(payload)
  await expect(item.locator('img, script')).toHaveCount(0)
  expect(dialogs).toBe(0)
})

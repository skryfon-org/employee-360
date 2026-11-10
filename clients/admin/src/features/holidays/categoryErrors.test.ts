import { describe, expect, it } from 'vitest'
import MockAdapter from 'axios-mock-adapter'
import { apiClient } from '@employee360/api-client'
import { isCategoryDuplicateError, isCategoryInUseError } from './categoryErrors'

async function failure(code: string, status = 409): Promise<unknown> {
  const mock = new MockAdapter(apiClient)
  mock.onGet('/x').reply(status, { success: false, error: { code, message: 'm' } })
  try {
    await apiClient.get('/x')
  } catch (e) {
    return e
  } finally {
    mock.restore()
  }
}

describe('category conflict classification', () => {
  it('matches CATEGORY_IN_USE as in-use only', async () => {
    const err = await failure('CATEGORY_IN_USE')
    expect(isCategoryInUseError(err)).toBe(true)
    expect(isCategoryDuplicateError(err)).toBe(false)
  })
  it('treats CONFLICT as a duplicate name, not in-use', async () => {
    const err = await failure('CONFLICT')
    expect(isCategoryDuplicateError(err)).toBe(true)
    expect(isCategoryInUseError(err)).toBe(false)
  })
  it('ignores other errors', async () => {
    expect(isCategoryInUseError(new Error('boom'))).toBe(false)
    expect(isCategoryDuplicateError(await failure('NOT_FOUND', 404))).toBe(false)
  })
})

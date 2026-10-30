import { describe, it, expect, vi, beforeEach } from 'vitest'
import { appRouter } from '../_app'
import { createTRPCContext } from '../../trpc'
import type { Session } from 'next-auth'

vi.mock('@/auth', () => ({
  auth: vi.fn(),
}))

import { auth } from '@/auth'
// auth is overloaded (session getter and middleware wrapper); tests use the getter.
const mockAuth = vi.mocked(auth as unknown as () => Promise<Session | null>)

function makeContext(session: Session | null = null) {
  const req = new Request('http://localhost/api/trpc') as import('next/server').NextRequest
  mockAuth.mockResolvedValue(session)
  return createTRPCContext({ req })
}

const validSession: Session = {
  user: { id: '123', email: 'test@example.com', name: 'Test User' },
  expires: '2099-01-01T00:00:00.000Z',
}

describe('protectedProcedure', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  it('throws UNAUTHORIZED when no session present', async () => {
    const ctx = await makeContext(null)
    const caller = appRouter.createCaller(ctx)
    await expect(caller.auth.session()).rejects.toMatchObject({ code: 'UNAUTHORIZED' })
  })

  it('allows access with valid session', async () => {
    const ctx = await makeContext(validSession)
    const caller = appRouter.createCaller(ctx)
    const result = await caller.auth.session()
    expect(result).toMatchObject({ authenticated: true, user: { email: 'test@example.com' } })
  })

  it('returns user data in session query', async () => {
    const ctx = await makeContext(validSession)
    const caller = appRouter.createCaller(ctx)
    const result = await caller.auth.session()
    expect(result.user?.name).toBe('Test User')
  })

  it('signOut returns success for authenticated user', async () => {
    const ctx = await makeContext(validSession)
    const caller = appRouter.createCaller(ctx)
    const result = await caller.auth.signOut()
    expect(result).toEqual({ success: true })
  })
})

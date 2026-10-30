import { it, expect } from 'vitest'
import { cn } from '@/lib/utils'

it('returns a single class unchanged', () => {
  expect(cn('foo')).toBe('foo')
})

it('merges multiple classes', () => {
  expect(cn('foo', 'bar')).toBe('foo bar')
})

it('handles undefined and falsy values', () => {
  expect(cn('foo', undefined, false, null, 'bar')).toBe('foo bar')
})

it('resolves tailwind conflicts in favour of the last class', () => {
  expect(cn('p-4', 'p-2')).toBe('p-2')
})

it('handles conditional object syntax', () => {
  expect(cn({ 'font-bold': true, italic: false })).toBe('font-bold')
})

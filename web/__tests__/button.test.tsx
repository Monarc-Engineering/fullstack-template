import { it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Button } from '@/components/ui/button'

it('renders with label', () => {
  render(<Button>Save</Button>)
  expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument()
})

it('renders as disabled when disabled prop is set', () => {
  render(<Button disabled>Submit</Button>)
  expect(screen.getByRole('button', { name: 'Submit' })).toBeDisabled()
})

it('applies variant via data-variant attribute', () => {
  render(<Button variant="destructive">Delete</Button>)
  expect(screen.getByRole('button', { name: 'Delete' })).toHaveAttribute(
    'data-variant',
    'destructive'
  )
})

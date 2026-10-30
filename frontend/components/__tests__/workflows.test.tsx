import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom/vitest'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { HomePage } from '../home-page'
import { FinePage } from '../fine-page'
import { DrivePage } from '../drive-page'

const stats = { today: 0, total: 0, filtered_count: 0, brands: [], vehicles: [] }
const jsonResponse = (body: unknown, ok = true) => ({ ok, json: async () => body })

beforeEach(() => vi.stubGlobal('fetch', vi.fn()))
afterEach(() => { cleanup(); vi.unstubAllGlobals() })

describe('incident report', () => {
  it('submits a selected incident and refreshes persisted statistics', async () => {
    const fetchMock = vi.mocked(fetch)
    fetchMock.mockResolvedValueOnce(jsonResponse(stats) as Response)
      .mockResolvedValueOnce(jsonResponse({ ok: true }) as Response)
      .mockResolvedValueOnce(jsonResponse({ ...stats, today: 1, total: 1 }) as Response)
    const user = userEvent.setup()
    render(<HomePage />)
    await screen.findAllByText('No incidents logged for this period yet.')
    await user.click(screen.getByRole('button', { name: /I got dazzled/i }))
    await user.selectOptions(screen.getByLabelText('Vehicle type'), 'Sedan')
    await user.selectOptions(screen.getByLabelText('Brand'), 'Audi')
    fireEvent.change(screen.getByRole('slider'), { target: { value: '5' } })
    await user.click(screen.getByRole('button', { name: /Add to the record/i }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3))
    expect(fetchMock.mock.calls[1]?.[0]).toBe('/api/incidents')
    expect(JSON.parse(String(fetchMock.mock.calls[1]?.[1]?.body))).toEqual({ vehicle_type: 'Sedan', brand: 'Audi', location: '', severity: 5, local_hour: new Date().getHours() })
    expect(await screen.findByRole('status')).toBeInTheDocument()
    expect(await screen.findAllByText('1')).toHaveLength(3)
  })

  it('shows a server error without closing the report form', async () => {
    const fetchMock = vi.mocked(fetch)
    fetchMock.mockResolvedValueOnce(jsonResponse(stats) as Response)
      .mockResolvedValueOnce(jsonResponse({ error: 'failed' }, false) as Response)
    const user = userEvent.setup()
    render(<HomePage />)
    await screen.findAllByText('No incidents logged for this period yet.')
    await user.click(screen.getByRole('button', { name: /I got dazzled/i }))
    await user.click(screen.getByRole('button', { name: /Add to the record/i }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Your report could not be saved')
    expect(screen.getByLabelText('Brand')).toBeInTheDocument()
  })

  it('filters the live rankings when a time period is selected', async () => {
    const fetchMock = vi.mocked(fetch)
    fetchMock.mockResolvedValueOnce(jsonResponse(stats) as Response)
      .mockResolvedValueOnce(jsonResponse({ ...stats, filtered_count: 1, brands: [{ name: 'BMW', value: 1 }], vehicles: [{ name: 'SUV', value: 1 }] }) as Response)
    const user = userEvent.setup()
    render(<HomePage />)
    await screen.findAllByText('No incidents logged for this period yet.')
    await user.click(screen.getByRole('button', { name: 'This week' }))
    expect(await screen.findByText('1 incidents logged')).toBeInTheDocument()
    expect(fetchMock.mock.calls[1]?.[0]).toBe('/api/stats?period=week')
    expect(screen.getByText('BMW')).toBeInTheDocument()
  })

  it('shows a distinct loading state while statistics are pending', () => {
    vi.mocked(fetch).mockReturnValueOnce(new Promise<Response>(() => {}))
    render(<HomePage />)
    expect(screen.getAllByText('Loading incidents…')).toHaveLength(2)
  })
})

describe('other existing workflows', () => {
  it('generates a citation from the entered vehicle description', async () => {
    const user = userEvent.setup()
    render(<FinePage />)
    await user.type(screen.getByPlaceholderText('e.g. A very confident SUV'), 'Blue sedan')
    await user.click(screen.getByRole('button', { name: /Generate citation/i }))
    expect(screen.getByText(/Blue sedan/)).toBeInTheDocument()
    expect(screen.getByText(/PENALTY/)).toBeInTheDocument()
  })

  it('renders the actual saved driving leaderboard response', async () => {
    vi.mocked(fetch).mockResolvedValueOnce(jsonResponse({ scores: [{ username: 'Night Owl', distance: 4.2 }] }) as Response)
    render(<DrivePage />)
    expect(await screen.findByText('Night Owl')).toBeInTheDocument()
    expect(screen.getByText('4.2 km')).toBeInTheDocument()
  })
})

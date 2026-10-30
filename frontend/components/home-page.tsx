'use client'

import { useEffect, useState } from 'react'
import { ArrowDown, ArrowUp, Minus } from 'lucide-react'
import { brands, confirmations, vehicleTypes } from '@/lib/content'

const sortedBrands = [...brands].sort((a, b) => a.localeCompare(b))
const sortedVehicleTypes = [...vehicleTypes].sort((a, b) => a.localeCompare(b))

type Ranking = { name: string; value: number }
type HourlyCount = { hour: number; count: number }
type Stats = { today: number; total: number; filtered_count: number; brands: Ranking[]; vehicles: Ranking[]; local_hours: HourlyCount[] }
const emptyStats: Stats = { today: 0, total: 0, filtered_count: 0, brands: [], vehicles: [], local_hours: Array.from({ length: 24 }, (_, hour) => ({ hour, count: 0 })) }
const reportCountStorageKey = 'dazzled.successful-reports'

function readPersonalReportCount() {
  try {
    const count = Number(window.localStorage.getItem(reportCountStorageKey))
    return Number.isSafeInteger(count) && count >= 0 ? count : 0
  } catch {
    return 0
  }
}

function personalReportTitle(count: number) {
  if (count === 0) return 'Fresh retinas'
  if (count < 3) return 'Nocturnal pedestrian'
  if (count < 6) return 'Beam survivor'
  if (count < 11) return 'Night shift veteran'
  return 'Legally blind (allegedly)'
}

function formatHour(hour: number) {
  return `${String(hour).padStart(2, '0')}:00`
}
const dailyExcuses = [
  'The road was too dark to see the road.',
  'The lamps were calibrated against the moon.',
  'The driver mistook dipped beam for a personal opinion.',
  'Our instruments classify this as “someone else’s problem.”',
  'The vehicle was merely attempting to contact low-flying aircraft.',
  'Visibility was excellent from inside that particular car.',
  'The department has forwarded the matter to a brighter department.',
]

async function requestStats(period: string): Promise<Stats> {
  const response = await fetch(`/api/stats?period=${period}`)
  if (!response.ok) throw new Error('Could not load incident statistics.')
  return await response.json() as Stats
}

function BarList({ items, loading, error }: { items: Ranking[]; loading: boolean; error: boolean }) {
  const max = Math.max(...items.map((item) => item.value), 1)
  return <div className="bar-list">{loading ? <p className="leaderboard-empty">Loading incidents…</p> : error ? <p className="leaderboard-empty">Statistics could not be loaded.</p> : items.length ? items.map((item, index) => <div className="bar-row" key={item.name}><div className="bar-label"><span>{String(index + 1).padStart(2, '0')}</span><strong>{item.name}</strong><b>{item.value}</b></div><div className="bar-track"><i style={{ width: `${Math.max(3, item.value / max * 100)}%` }} /></div></div>) : <p className="leaderboard-empty">No incidents logged for this period yet.</p>}</div>
}

export function HomePage() {
  const [open, setOpen] = useState(false)
  const [severity, setSeverity] = useState(3)
  const [brand, setBrand] = useState('BMW')
  const [vehicleType, setVehicleType] = useState('SUV / crossover')
  const [location, setLocation] = useState('')
  const [localHour, setLocalHour] = useState(0)
  const [personalReportCount, setPersonalReportCount] = useState(0)
  const [stats, setStats] = useState<Stats>(emptyStats)
  const [bulletinDate, setBulletinDate] = useState('')
  const [confirmation, setConfirmation] = useState('')
  const [tab, setTab] = useState('today')
  const [loading, setLoading] = useState(true)
  const [statsError, setStatsError] = useState(false)
  const [formError, setFormError] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    const updateDate = () => setBulletinDate(new Date().toISOString().slice(0, 10))
    const initialTimer = window.setTimeout(updateDate, 0)
    const timer = window.setInterval(updateDate, 60_000)
    return () => {
      window.clearTimeout(initialTimer)
      window.clearInterval(timer)
    }
  }, [])

  useEffect(() => {
    const timer = window.setTimeout(() => setPersonalReportCount(readPersonalReportCount()), 0)
    return () => window.clearTimeout(timer)
  }, [])

  useEffect(() => {
    let active = true
    void requestStats(tab)
      .then((result) => { if (active) setStats(result) })
      .catch(() => { if (active) setStatsError(true) })
      .finally(() => { if (active) setLoading(false) })
    return () => { active = false }
  }, [tab])

  async function submit(e: React.FormEvent) {
    e.preventDefault(); setSaving(true); setFormError('')
    try {
      const response = await fetch('/api/incidents', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ vehicle_type: vehicleType, brand, location, severity, local_hour: localHour }) })
      if (!response.ok) throw new Error('Your report could not be saved. Please try again.')
      const nextPersonalReportCount = Math.max(personalReportCount, readPersonalReportCount()) + 1
      setPersonalReportCount(nextPersonalReportCount)
      try { window.localStorage.setItem(reportCountStorageKey, String(nextPersonalReportCount)) } catch { /* Keep the session-only count if browser storage is unavailable. */ }
      setConfirmation(confirmations[Math.floor(Math.random() * confirmations.length)])
      setOpen(false)
      setLoading(true)
      setStatsError(false)
      void requestStats(tab)
        .then(setStats)
        .catch(() => setStatsError(true))
        .finally(() => setLoading(false))
    } catch (reason) { setFormError(reason instanceof Error ? reason.message : 'Your report could not be saved.') }
    finally { setSaving(false) }
  }

  const dateNumber = bulletinDate ? Math.floor(Date.parse(`${bulletinDate}T00:00:00Z`) / 86_400_000) : 0
  const dailyExcuse = dailyExcuses[dateNumber % dailyExcuses.length]
  const periodLabel = tab === 'today' ? 'today' : tab === 'week' ? 'this week' : 'all time'
  const localHours = stats.local_hours ?? emptyStats.local_hours
  const busiestLocalHourCount = Math.max(...localHours.map(({ count }) => count), 0)
  const timedReportCount = localHours.reduce((total, { count }) => total + count, 0)

  return <>
    <main className="main-content home-page">
      <section className="hero-grid">
        <div className="hero-copy"><p className="eyebrow"><span className="signal-dot" /> A public service announcement</p><h1>Modern headlights.<br /><em>Ancient problem.</em></h1><p className="hero-tagline">Regulators are still studying the problem.<br />We&apos;re already counting.</p><button className="dazzle-button" onClick={() => { setLocalHour(new Date().getHours()); setOpen(true); setFormError('') }}><span>I got dazzled</span><ArrowDown aria-hidden="true" /></button>{confirmation && <p className="confirmation" role="status">{confirmation}</p>}</div>
        <div className="hero-aside"><span className="aside-number">01</span><p>Night driving has become a trust exercise between your eyes and somebody else&apos;s bulb alignment.</p><span className="aside-rule" /></div>
      </section>
      {open && <form className="report-panel" onSubmit={submit}><div className="panel-heading"><span>Report a new incident</span><button type="button" onClick={() => setOpen(false)} aria-label="Close report">×</button></div><div className="field-grid"><label>Vehicle type<select value={vehicleType} onChange={(e) => setVehicleType(e.target.value)}>{sortedVehicleTypes.map((v) => <option key={v}>{v}</option>)}</select></label><label>Brand<select value={brand} onChange={(e) => setBrand(e.target.value)}>{sortedBrands.map((v) => <option key={v}>{v}</option>)}</select></label></div><label className="location-label">Approximate time it happened (your local time)<select value={localHour} onChange={(e) => setLocalHour(Number(e.target.value))}>{Array.from({ length: 24 }, (_, hour) => <option key={hour} value={hour}>{formatHour(hour)}</option>)}</select><small>Choose the closest hour. This is used for the Peak Blindness chart.</small></label><label className="location-label">Where did it happen? <input maxLength={100} value={location} onChange={(e) => setLocation(e.target.value)} placeholder="City, road, or nearby landmark (optional)" /><small>Optional place text, max 100 characters. Stored with the report, visible to site operators, and not shown publicly. No GPS is collected; please avoid exact addresses.</small></label><label className="range-label">Severity <span>{severity} / 5</span><input type="range" min="1" max="5" value={severity} onChange={(e) => setSeverity(Number(e.target.value))} /><small><span>mildly annoyed</span><span>saw my ancestors</span></small></label>{formError && <p className="leaderboard-error" role="alert">{formError}</p>}<button className="submit-button" type="submit" disabled={saving}>{saving ? 'Saving report…' : 'Add to the record'} <ArrowDown aria-hidden="true" /></button></form>}
      <section className="stats-strip"><div><span>Dazzled today</span><strong>{loading ? '—' : stats.today.toLocaleString()}</strong><small><ArrowUp aria-hidden="true" /> Reports logged today</small></div><div><span>Dazzled all time</span><strong>{loading ? '—' : stats.total.toLocaleString()}</strong><small><Minus aria-hidden="true" /> Across all reports</small></div><div className="personal-meter"><span>Your Retinal Damage Meter</span><strong>{personalReportCount}</strong><small>{personalReportTitle(personalReportCount)} · this browser only</small></div></section>
      <section className="peak-blindness" aria-labelledby="peak-blindness-title">
        <div className="peak-heading"><div><p className="eyebrow">The timing of the crime</p><h2 id="peak-blindness-title">Peak blindness</h2></div><p>All time, by the approximate local hour each report says it happened.</p></div>
        {statsError ? <p className="leaderboard-empty">The department&apos;s hourly records are unavailable.</p> : timedReportCount < 30 ? <p className="leaderboard-empty">Not enough reports yet. The department requires 30 timed reports before it publishes a pattern ({timedReportCount} / 30 so far).</p> : <>
          <div className="hour-chart" role="img" aria-label={`All-time incident reports by local hour. ${localHours.map(({ hour, count }) => `${formatHour(hour)}: ${count}`).join(', ')}`}>
            {localHours.map(({ hour, count }) => <div className="hour-column" key={hour} title={`${formatHour(hour)} — ${count} report${count === 1 ? '' : 's'}`}><i style={{ height: `${busiestLocalHourCount ? Math.max(3, count / busiestLocalHourCount * 100) : 0}%` }} /><span>{hour % 6 === 0 ? formatHour(hour) : ''}</span></div>)}
          </div>
          <p className="peak-footnote">Reports filed before local-time tracking was added aren&apos;t included.</p>
        </>}
      </section>
      <section className="night-bulletin" aria-labelledby="bulletin-title">
        <div className="bulletin-stamp">
          <p className="eyebrow">Night shift bulletin</p>
          <span>DISPATCH {bulletinDate || 'TODAY'}</span>
        </div>
        <div className="bulletin-suspect">
          <p className="eyebrow">The leading suspect</p>
          {loading ? <strong>Checking the files…</strong> : statsError ? <strong>Records unavailable</strong> : stats.vehicles.length ? <><strong>{stats.vehicles[0]!.name}</strong><small>{stats.vehicles[0]!.value} report{stats.vehicles[0]!.value === 1 ? '' : 's'} · {periodLabel}</small></> : <><strong>No suspect yet</strong><small>The department is still waiting for its first witness.</small></>}
        </div>
        <div className="bulletin-excuse">
          <p className="eyebrow">Official explanation</p>
          <h2 id="bulletin-title">{dailyExcuse}</h2>
        </div>
      </section>
      <section className="shame-section"><div className="section-heading"><div><p className="eyebrow">The evidence</p><h2>Hall of shame</h2></div><div className="tabs">{[{label:'Today', value:'today'}, {label:'This week', value:'week'}, {label:'All time', value:'all'}].map((item) => <button key={item.value} className={tab === item.value ? 'active' : ''} onClick={() => { if (tab === item.value) return; setLoading(true); setStatsError(false); setTab(item.value) }}>{item.label}</button>)}</div></div>{statsError && <p className="leaderboard-error" role="alert">Incident statistics are temporarily unavailable.</p>}<div className="charts-grid"><div className="chart-card"><div className="chart-title"><span>By brand</span><small>{loading ? 'Loading…' : `${stats.filtered_count} incidents logged`}</small></div><BarList items={stats.brands} loading={loading} error={statsError} /></div><div className="chart-card"><div className="chart-title"><span>By vehicle type</span><small>Same darkness, different shape</small></div><BarList items={stats.vehicles} loading={loading} error={statsError} /></div></div></section>
    </main>
  </>
}

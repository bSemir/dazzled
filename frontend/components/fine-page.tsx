// Generate and share satirical citations locally in the browser.
'use client'

import { useState } from 'react'
import { Copy, Download, RefreshCw } from 'lucide-react'
import { penalties, situations } from '@/lib/content'
import { downloadCard } from '@/lib/download-card'

export function FinePage() {
  const [vehicle, setVehicle] = useState('')
  const [situation, setSituation] = useState(situations[0])
  const [fine, setFine] = useState(0)
  const issue = () => setFine(Math.floor(Math.random() * penalties.length))
  const description = vehicle || 'a suspiciously bright object'
  const citation = `THE DEPARTMENT OF THINGS WE CANNOT SEE\nCitation no. DZ-${String(48291 + fine).padStart(6, '0')}\n\nThe vehicle described as “${description}” was observed during: ${situation.toLowerCase()}.\n\nPENALTY\n${penalties[fine]}`

  function copyCitation() {
    void navigator.clipboard?.writeText(citation)
  }

  function downloadCitation() {
    downloadCard(
      'Citation issued',
      `${description} — ${situation.toLowerCase()}. Penalty: ${penalties[fine]}`,
      'dazzled-citation',
    )
  }

  return (
    <main className="main-content fine-page">
      <section className="page-intro">
        <p className="eyebrow">Official-ish paperwork / 02</p>
        <h1>Issue a fine.<br /><em>Feel slightly better.</em></h1>
        <p>Because a formal document is the natural response to a headlight aimed directly at your soul.</p>
      </section>

      <div className="fine-layout">
        <form className="fine-form" onSubmit={(event) => { event.preventDefault(); issue() }}>
          <label>
            Describe the vehicle
            <input
              maxLength={60}
              value={vehicle}
              onChange={(event) => setVehicle(event.target.value)}
              placeholder="e.g. A very confident SUV"
            />
          </label>
          <label>
            Situation
            <select value={situation} onChange={(event) => setSituation(event.target.value)}>
              {situations.map((item) => <option key={item}>{item}</option>)}
            </select>
          </label>
          <button className="submit-button" type="submit">
            Generate citation <RefreshCw aria-hidden="true" />
          </button>
          <p className="form-note">This is satire. The fine is not legally enforceable, regrettably.</p>
        </form>

        <div className="citation-wrap">
          <article className="citation" aria-live="polite">
            <div className="citation-top">
              <span>● Department form 04-B</span>
              <span>VOID / NOT REAL</span>
            </div>
            <div className="citation-seal">D</div>
            <p className="citation-dept">The Department of<br />Things We Cannot See</p>
            <div className="citation-meta">
              <span>CASE DZ-{String(48291 + fine).padStart(6, '0')}</span>
              <span>DATE {new Date().toLocaleDateString('en-GB')}</span>
            </div>
            <div className="citation-body">
              <small>OFFENCE</small>
              <p>Excessive luminance in a shared public environment.</p>
              <small>DESCRIPTION</small>
              <p>“{description}” — {situation.toLowerCase()}.</p>
              <small>PENALTY</small>
              <p className="penalty">{penalties[fine]}</p>
            </div>
            <div className="citation-signature">
              <span>Issued by<br /><strong>Office of Nighttime Frustration</strong></span>
              <span className="signature">D. Regulator</span>
            </div>
          </article>
          <div className="citation-actions">
            <button onClick={copyCitation}><Copy aria-hidden="true" /> Copy text</button>
            <button onClick={downloadCitation}><Download aria-hidden="true" /> Download as image</button>
          </div>
        </div>
      </div>
    </main>
  )
}

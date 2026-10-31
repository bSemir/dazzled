// Run the canvas driving game, audio controls, and score submission in the browser.
'use client'

import { useEffect, useRef, useState } from 'react'
import Link from 'next/link'
import { ArrowLeft, ArrowRight, Download, RotateCcw } from 'lucide-react'
import { driveLines } from '@/lib/content'
import { downloadCard } from '@/lib/download-card'

type Score = { username: string; distance: number }
type Horn = { context: AudioContext; gain: GainNode; oscillators: OscillatorNode[] }
type Music = { context: AudioContext; gain: GainNode; timer: number; oscillators: Set<OscillatorNode> }

function getAudioContext(contextRef: { current: AudioContext | null }) {
  const AudioContextConstructor = window.AudioContext
  if (!AudioContextConstructor) return null
  const context = contextRef.current ?? new AudioContextConstructor()
  contextRef.current = context
  void context.resume()
  return context
}

function startMusic(contextRef: { current: AudioContext | null }, musicRef: { current: Music | null }) {
  if (musicRef.current) return
  const context = getAudioContext(contextRef)
  if (!context) return
  const gain = context.createGain()
  gain.gain.setValueAtTime(0.0001, context.currentTime)
  gain.gain.linearRampToValueAtTime(0.42, context.currentTime + 0.8)
  gain.connect(context.destination)
  const music: Music = { context, gain, timer: 0, oscillators: new Set() }
  musicRef.current = music
  const notes = [261.63, 329.63, 392, 329.63, 293.66, 349.23, 440, 349.23]
  let noteIndex = 0
  const playNote = () => {
    const now = context.currentTime
    const oscillator = context.createOscillator()
    const envelope = context.createGain()
    oscillator.type = 'sine'
    oscillator.frequency.setValueAtTime(notes[noteIndex % notes.length]!, now)
    noteIndex += 1
    envelope.gain.setValueAtTime(0.0001, now)
    envelope.gain.linearRampToValueAtTime(0.035, now + 0.08)
    envelope.gain.setValueAtTime(0.035, now + 0.32)
    envelope.gain.exponentialRampToValueAtTime(0.0001, now + 0.68)
    oscillator.connect(envelope)
    envelope.connect(gain)
    music.oscillators.add(oscillator)
    oscillator.onended = () => {
      oscillator.disconnect()
      envelope.disconnect()
      music.oscillators.delete(oscillator)
    }
    oscillator.start(now)
    oscillator.stop(now + 0.7)
  }
  playNote()
  music.timer = window.setInterval(playNote, 680)
}

function stopMusic(musicRef: { current: Music | null }) {
  const music = musicRef.current
  if (!music) return
  musicRef.current = null
  window.clearInterval(music.timer)
  const now = music.context.currentTime
  music.gain.gain.cancelScheduledValues(now)
  music.gain.gain.setValueAtTime(music.gain.gain.value, now)
  music.gain.gain.linearRampToValueAtTime(0.0001, now + 0.25)
  for (const oscillator of music.oscillators) oscillator.stop(now + 0.26)
  window.setTimeout(() => music.gain.disconnect(), 350)
}

function playCrash(contextRef: { current: AudioContext | null }) {
  const context = getAudioContext(contextRef)
  if (!context) return
  const now = context.currentTime
  const noiseBuffer = context.createBuffer(1, Math.floor(context.sampleRate * 0.2), context.sampleRate)
  const noiseData = noiseBuffer.getChannelData(0)
  for (let index = 0; index < noiseData.length; index += 1) noiseData[index] = Math.random() * 2 - 1
  const noise = context.createBufferSource()
  const filter = context.createBiquadFilter()
  const noiseGain = context.createGain()
  noise.buffer = noiseBuffer
  filter.type = 'lowpass'
  filter.frequency.setValueAtTime(700, now)
  noiseGain.gain.setValueAtTime(0.09, now)
  noiseGain.gain.exponentialRampToValueAtTime(0.0001, now + 0.2)
  noise.connect(filter)
  filter.connect(noiseGain)
  noiseGain.connect(context.destination)
  noise.start(now)
  noise.stop(now + 0.21)
  const thump = context.createOscillator()
  const thumpGain = context.createGain()
  thump.type = 'sine'
  thump.frequency.setValueAtTime(105, now)
  thump.frequency.exponentialRampToValueAtTime(48, now + 0.2)
  thumpGain.gain.setValueAtTime(0.16, now)
  thumpGain.gain.exponentialRampToValueAtTime(0.0001, now + 0.2)
  thump.connect(thumpGain)
  thumpGain.connect(context.destination)
  thump.start(now)
  thump.stop(now + 0.21)
}

function startHorn(contextRef: { current: AudioContext | null }, hornRef: { current: Horn | null }) {
  if (hornRef.current) return
  const context = getAudioContext(contextRef)
  if (!context) return
  const now = context.currentTime
  const gain = context.createGain()
  gain.gain.setValueAtTime(0.0001, now)
  gain.gain.linearRampToValueAtTime(0.055, now + 0.035)
  gain.connect(context.destination)
  const oscillators: OscillatorNode[] = []
  for (const frequency of [390, 490]) {
    const oscillator = context.createOscillator()
    oscillator.type = 'triangle'
    oscillator.frequency.setValueAtTime(frequency, now)
    oscillator.connect(gain)
    oscillator.start(now)
    oscillators.push(oscillator)
  }
  hornRef.current = { context, gain, oscillators }
}

function stopHorn(hornRef: { current: Horn | null }) {
  const horn = hornRef.current
  if (!horn) return
  hornRef.current = null
  const now = horn.context.currentTime
  horn.gain.gain.cancelScheduledValues(now)
  horn.gain.gain.setValueAtTime(horn.gain.gain.value, now)
  horn.gain.gain.linearRampToValueAtTime(0.0001, now + 0.08)
  for (const oscillator of horn.oscillators) oscillator.stop(now + 0.09)
  window.setTimeout(() => horn.gain.disconnect(), 150)
}

async function getScores(): Promise<Score[]> {
  const response = await fetch('/api/leaderboard')
  if (!response.ok) throw new Error('Could not load the scoreboard.')
  const data = await response.json() as { scores: Score[] }
  return data.scores
}

export function DrivePage() {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const audioContextRef = useRef<AudioContext | null>(null)
  const hornRef = useRef<Horn | null>(null)
  const musicRef = useRef<Music | null>(null)
  const distanceRef = useRef(0)
  const [started, setStarted] = useState(false)
  const [gameOver, setGameOver] = useState(false)
  const [distance, setDistance] = useState(0)
  const [bestDistance, setBestDistance] = useState(0)
  const [flashCount, setFlashCount] = useState(0)
  const [username, setUsername] = useState('')
  const [scores, setScores] = useState<Score[]>([])
  const [gameToken, setGameToken] = useState('')
  const [startingGame, setStartingGame] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const [leaderboardError, setLeaderboardError] = useState('')
  const [scoresLoaded, setScoresLoaded] = useState(false)
  const [gameOverMessage, setGameOverMessage] = useState('')
  const [savingScore, setSavingScore] = useState(false)

  useEffect(() => {
    let active = true
    void getScores()
      .then((result) => { if (active) setScores(result) })
      .catch(() => { if (active) setLeaderboardError('The scoreboard is hiding in the glovebox.') })
      .finally(() => { if (active) setScoresLoaded(true) })
    return () => { active = false }
  }, [])

  useEffect(() => () => {
    stopHorn(hornRef)
    stopMusic(musicRef)
    void audioContextRef.current?.close()
  }, [])

  useEffect(() => {
    if (!started || gameOver) return
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    let lane = 1
    let flashStartedAt = 0
    let dazzleUntil = 0
    let spaceHeld = false
    let frame = 0
    // Normalize movement to 60 Hz so refresh rate does not change game difficulty.
    let lastFrameAt = performance.now()
    let nextFlashFrame = 45
    let raf = 0
    let ended = false
    const cars = [
      { lane: Math.floor(Math.random() * 3), y: -120, color: '#eafaff', speed: 1 },
      { lane: (Math.floor(Math.random() * 3) + 1) % 3, y: -420, color: '#ffb6a6', speed: 0.92 },
    ]
    const resize = () => {
      canvas.width = canvas.clientWidth * 2
      canvas.height = canvas.clientHeight * 2
      ctx.setTransform(2, 0, 0, 2, 0, 0)
    }
    resize()
    window.addEventListener('resize', resize)
    const releaseSpace = () => {
      if (!spaceHeld) return
      spaceHeld = false
      dazzleUntil = performance.now() + 180
      stopHorn(hornRef)
    }
    const keyUp = (event: KeyboardEvent) => { if (event.code === 'Space') releaseSpace() }
    const moveLane = (event: KeyboardEvent) => {
      if (event.key === 'ArrowLeft') { event.preventDefault(); lane = Math.max(0, lane - 1) }
      if (event.key === 'ArrowRight') { event.preventDefault(); lane = Math.min(2, lane + 1) }
      if (event.code === 'Space') {
        event.preventDefault()
        spaceHeld = true
        if (!event.repeat) startHorn(audioContextRef, hornRef)
      }
    }
    window.addEventListener('keydown', moveLane)
    window.addEventListener('keyup', keyUp)
    window.addEventListener('blur', releaseSpace)
    const endGame = () => {
      if (ended) return
      ended = true
      stopMusic(musicRef)
      stopHorn(hornRef)
      playCrash(audioContextRef)
      setGameOverMessage(driveLines[Math.floor(Math.random() * driveLines.length)].replace('{distance}', distanceRef.current.toFixed(1)))
      setGameOver(true)
    }
    const draw = () => {
      if (ended) return
      const frameTime = performance.now()
      const deltaFrames = Math.min(2, Math.max(0, (frameTime - lastFrameAt) * 60 / 1000))
      lastFrameAt = frameTime
      const width = canvas.clientWidth
      const height = canvas.clientHeight
      const speed = 8.5 + Math.min(9, distanceRef.current * 0.6)
      const roadWidthAt = (y: number) => width * (0.36 + 0.42 * Math.min(1, Math.max(0, y / height)))
      const roadLeftAt = (y: number) => (width - roadWidthAt(y)) / 2
      const laneCenter = (whichLane: number, y: number) => roadLeftAt(y) + roadWidthAt(y) * ((whichLane + 0.5) / 3)
      ctx.fillStyle = '#06080a'
      ctx.fillRect(0, 0, width, height)
      ctx.fillStyle = '#111b20'
      ctx.beginPath()
      ctx.moveTo(roadLeftAt(0), 0)
      ctx.lineTo(roadLeftAt(height), height)
      ctx.lineTo(roadLeftAt(height) + roadWidthAt(height), height)
      ctx.lineTo(roadLeftAt(0) + roadWidthAt(0), 0)
      ctx.closePath()
      ctx.fill()
      ctx.strokeStyle = '#263239'
      ctx.lineWidth = 2
      for (const edge of [0, 1]) {
        ctx.beginPath()
        ctx.moveTo(roadLeftAt(0) + roadWidthAt(0) * edge, 0)
        ctx.lineTo(roadLeftAt(height) + roadWidthAt(height) * edge, height)
        ctx.stroke()
      }
      ctx.strokeStyle = '#34434a'
      ctx.lineWidth = 1.5
      ctx.setLineDash([18, 20])
      ctx.lineDashOffset = -((frame * speed * 0.7) % 38)
      for (const divider of [1, 2]) {
        ctx.beginPath()
        ctx.moveTo(roadLeftAt(0) + roadWidthAt(0) * (divider / 3), 0)
        ctx.lineTo(roadLeftAt(height) + roadWidthAt(height) * (divider / 3), height)
        ctx.stroke()
      }
      ctx.setLineDash([])
      ctx.lineDashOffset = 0
      const drawCar = (centerX: number, top: number, color: string, scale: number) => {
        const carWidth = 28 * scale
        const carHeight = 48 * scale
        const left = centerX - carWidth / 2
        ctx.fillStyle = '#030405'
        for (const wheelY of [top + 9 * scale, top + carHeight - 17 * scale]) {
          ctx.fillRect(left - 2.5 * scale, wheelY, 5 * scale, 9 * scale)
          ctx.fillRect(left + carWidth - 2.5 * scale, wheelY, 5 * scale, 9 * scale)
        }
        ctx.fillStyle = color
        ctx.shadowColor = color
        ctx.shadowBlur = 14 * scale
        ctx.beginPath()
        ctx.roundRect(left, top, carWidth, carHeight, 6 * scale)
        ctx.fill()
        ctx.shadowBlur = 0
        ctx.fillStyle = '#26343a'
        ctx.fillRect(left + 4 * scale, top + 12 * scale, carWidth - 8 * scale, 18 * scale)
        ctx.fillStyle = '#dff7ff'
        ctx.fillRect(left + 4 * scale, top + 3 * scale, 6 * scale, 2 * scale)
        ctx.fillRect(left + carWidth - 10 * scale, top + 3 * scale, 6 * scale, 2 * scale)
      }
      const playerTop = height - 92
      const playerHeight = 48
      const playerX = laneCenter(lane, playerTop + playerHeight / 2)
      if (spaceHeld || performance.now() < dazzleUntil) {
        const beamLength = playerHeight * 2
        const carWidth = 28
        const beam = ctx.createLinearGradient(playerX, playerTop - beamLength, playerX, playerTop)
        beam.addColorStop(0, 'rgba(234, 250, 255, 0.03)')
        beam.addColorStop(1, 'rgba(234, 250, 255, 0.36)')
        ctx.fillStyle = beam
        ctx.beginPath()
        ctx.moveTo(playerX - carWidth / 2, playerTop)
        ctx.lineTo(playerX + carWidth / 2, playerTop)
        ctx.lineTo(playerX + carWidth, playerTop - beamLength)
        ctx.lineTo(playerX - carWidth, playerTop - beamLength)
        ctx.closePath()
        ctx.fill()
      }
      drawCar(playerX, playerTop, '#b8eaff', 1)
      for (const car of cars) {
        car.y += speed * car.speed * deltaFrames
        if (car.y > height + 60) {
          car.y = -150 - Math.random() * 260
          car.lane = Math.floor(Math.random() * 3)
          const nextDistance = Number((distanceRef.current + 0.1).toFixed(1))
          distanceRef.current = nextDistance
          setDistance(nextDistance)
          setBestDistance((best) => Math.max(best, nextDistance))
        }
        const carScale = 0.65 + 0.35 * Math.min(1, Math.max(0, car.y / height))
        const carHeight = 48 * carScale
        drawCar(laneCenter(car.lane, car.y + carHeight / 2), car.y, car.color, carScale)
        if (car.lane === lane && car.y + carHeight >= playerTop && car.y <= playerTop + playerHeight) { endGame(); return }
      }
      if (frame >= nextFlashFrame) {
        flashStartedAt = performance.now()
        setFlashCount((count) => count + 1)
        nextFlashFrame = frame + Math.max(38, 92 - Math.floor(distanceRef.current * 3))
      }
      const flashElapsed = performance.now() - flashStartedAt
      if (flashStartedAt > 0 && flashElapsed < 600) {
        const flashOpacity = (1 - flashElapsed / 600) * 0.9
        ctx.fillStyle = `rgba(255,255,255,${flashOpacity})`
        ctx.fillRect(0, 0, width, height)
      }
      frame += deltaFrames
      raf = requestAnimationFrame(draw)
    }
    raf = requestAnimationFrame(draw)
    return () => {
      cancelAnimationFrame(raf)
      window.removeEventListener('resize', resize)
      window.removeEventListener('keydown', moveLane)
      window.removeEventListener('keyup', keyUp)
      window.removeEventListener('blur', releaseSpace)
      stopHorn(hornRef)
      stopMusic(musicRef)
    }
  }, [started, gameOver])

  async function start() {
    getAudioContext(audioContextRef)
    setStartingGame(true)
    setLeaderboardError('')
    try {
      // A fresh server-issued token binds the eventual score to this run's start time.
      const response = await fetch('/api/leaderboard/start', { method: 'POST' })
      if (!response.ok) throw new Error('A verified run could not be started. Please try again.')
      const data = await response.json() as { game_token: string }
      if (!data.game_token) throw new Error('A verified run could not be started. Please try again.')
      setGameToken(data.game_token)
      distanceRef.current = 0
      setDistance(0)
      setFlashCount(0)
      setGameOver(false)
      setGameOverMessage('')
      setSubmitted(false)
      setStarted(true)
      startMusic(audioContextRef, musicRef)
    } catch (error) {
      setLeaderboardError(error instanceof Error ? error.message : 'A verified run could not be started. Please try again.')
    } finally {
      setStartingGame(false)
    }
  }

  async function submitScore(event: React.FormEvent) {
    event.preventDefault()
    setLeaderboardError('')
    setSavingScore(true)
    try {
      if (!gameToken) throw new Error('This run has no verification token. Start a new run to post a score.')
      const response = await fetch('/api/leaderboard', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, distance, game_token: gameToken }),
      })
      if (!response.ok) {
        const body = await response.json().catch(() => ({})) as { error?: string }
        throw new Error(body.error ?? 'The scoreboard could not verify this run. Please start a new run.')
      }
      setSubmitted(true)
      setGameToken('')
      setUsername('')
      setScores(await getScores())
    } catch (error) {
      setLeaderboardError(error instanceof Error ? error.message : 'The score could not be saved. Try again.')
    } finally { setSavingScore(false) }
  }

  return (
    <main className="main-content drive-page">
      <div className="drive-heading">
        <div><p className="eyebrow">A public safety experiment / 03</p><h1>Blind <em>drive.</em></h1></div>
        <div className="drive-score"><span>Scoreboard</span><strong>{distance.toFixed(1)} <small>km</small></strong><div className="drive-stats"><span>Best {bestDistance.toFixed(1)} km</span><span>Flashes {flashCount}</span><span>Speed {Math.round(62 + Math.min(70, distance * 4.5))} km/h</span></div></div>
      </div>
      <div className="game-wrap">
        <canvas ref={canvasRef} aria-label="Blind drive game area" onTouchStart={(event) => {
          const x = event.touches[0]?.clientX ?? 0
          const rect = event.currentTarget.getBoundingClientRect()
          window.dispatchEvent(new KeyboardEvent('keydown', { key: x < rect.left + rect.width / 2 ? 'ArrowLeft' : 'ArrowRight' }))
        }} />
        {!started && <div className="game-overlay"><p className="eyebrow">Passenger mode only</p><h2>How far can you go<br />without seeing?</h2><p>For passengers only. Do not play while driving.</p><button className="submit-button" onClick={start} disabled={startingGame}>{startingGame ? 'Starting a verified run…' : 'Start driving'} <ArrowRight aria-hidden="true" /></button></div>}
        {gameOver && <div className="game-overlay"><p className="eyebrow">Visibility ended</p><h2>That was a close one.</h2><p>{gameOverMessage}</p><div className="overlay-actions"><button className="submit-button" onClick={start} disabled={startingGame}><RotateCcw aria-hidden="true" /> {startingGame ? 'Starting…' : 'Play again'}</button><button className="ghost-button" onClick={() => downloadCard('Blind drive', `Survived ${distance.toFixed(1)} km. Passenger mode only.`, 'dazzled-drive-score')}><Download aria-hidden="true" /> Download share card</button><Link href="/" className="text-link">Back to home</Link></div></div>}
      </div>
      <div className="drive-controls"><span><ArrowLeft aria-hidden="true" /> left</span><span>space: horn + high beams</span><span>right <ArrowRight aria-hidden="true" /></span></div>
      <section className="leaderboard-panel" aria-labelledby="leaderboard-title">
        <div className="leaderboard-title"><h2 id="leaderboard-title">Hall of almost driving</h2><p>public / time-checked / deeply prestigious</p></div>
        {gameOver && !submitted && <form className="leaderboard-form" onSubmit={submitScore}><input value={username} onChange={(event) => setUsername(event.target.value)} placeholder="Your alias, road goblin" aria-label="Leaderboard username" maxLength={20} required /><button type="submit" disabled={savingScore}>{savingScore ? 'Posting…' : `Post ${distance.toFixed(1)} km`}</button></form>}
        {submitted && <p className="leaderboard-empty" role="status">Your score is now haunting the internet.</p>}
        {leaderboardError && <p className="leaderboard-error" role="alert">{leaderboardError}</p>}
        <div className="leaderboard-list">{!scoresLoaded ? <p className="leaderboard-empty">Loading scoreboard…</p> : scores.length ? scores.map((score, index) => <div className="leaderboard-row" key={`${score.username}-${index}`}><span>#{index + 1}</span><strong>{score.username}</strong><span>{score.distance.toFixed(1)} km</span></div>) : <p className="leaderboard-empty">{leaderboardError ? 'The scoreboard could not be loaded.' : 'No scores yet. Be the first questionable legend.'}</p>}</div>
      </section>
    </main>
  )
}

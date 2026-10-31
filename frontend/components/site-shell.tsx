// Provide the navigation and footer shared by application pages.
import Link from 'next/link'

export function SiteShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="site-shell">
      <header className="site-header">
        <Link href="/" className="wordmark" aria-label="Dazzled home">
          <span className="wordmark-mark" />DAZZLED
        </Link>
        <nav aria-label="Primary navigation" className="site-nav">
          <Link href="/fine">Fine generator</Link>
          <Link href="/drive">Blind drive</Link>
        </nav>
      </header>
      {children}
      <footer className="site-footer">
        <div className="footer-links">
          <Link href="/">Home</Link>
          <Link href="/fine">Fine generator</Link>
          <Link href="/drive">Blind drive</Link>
        </div>
        <p>Satire. Not affiliated with any car manufacturer, and sadly not with any regulator either.</p>
      </footer>
    </div>
  )
}

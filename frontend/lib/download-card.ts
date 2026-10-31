// Create shareable SVG cards locally without sending their text to a server.
export function downloadCard(title: string, detail: string, filename: string) {
  const escapeXML = (value: string) => value.replace(/[<>&"']/g, (character) => ({ '<': '&lt;', '>': '&gt;', '&': '&amp;', '"': '&quot;', "'": '&apos;' })[character]!)
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630"><rect width="1200" height="630" fill="#0a0a0a"/><circle cx="1040" cy="90" r="190" fill="#cbff39" opacity=".08"/><text x="72" y="84" fill="#cbff39" font-family="Arial,sans-serif" font-size="20" letter-spacing="5">DAZZLED / PUBLIC RECORD</text><text x="72" y="280" fill="#f2f0e9" font-family="Arial,sans-serif" font-size="68" font-weight="700">${escapeXML(title)}</text><foreignObject x="72" y="330" width="1050" height="190"><div xmlns="http://www.w3.org/1999/xhtml" style="color:#a7aaa7;font:28px Arial,sans-serif;line-height:1.4">${escapeXML(detail)}</div></foreignObject><path d="M72 560h1056" stroke="#333"/><text x="72" y="600" fill="#777" font-family="Arial,sans-serif" font-size="18">SATIRE — NOT AN OFFICIAL DOCUMENT</text></svg>`
  const url = URL.createObjectURL(new Blob([svg], { type: 'image/svg+xml;charset=utf-8' }))
  const link = document.createElement('a')
  link.href = url
  link.download = filename.endsWith('.svg') ? filename : `${filename}.svg`
  link.click()
  window.setTimeout(() => URL.revokeObjectURL(url), 1000)
}

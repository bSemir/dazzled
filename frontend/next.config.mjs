// Configure Next.js to emit the static site consumed by the Go server.
/** @type {import('next').NextConfig} */
const nextConfig = {
  output: process.env.NODE_ENV === 'production' ? 'export' : undefined,
  ...(process.env.NODE_ENV !== 'production' ? {
    async rewrites() {
      return [{ source: '/api/:path*', destination: `${process.env.API_ORIGIN || 'http://localhost:8080'}/api/:path*` }]
    },
  } : {}),
  images: {
    unoptimized: true,
  },
}

export default nextConfig

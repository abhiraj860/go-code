import type { NextConfig } from 'next';

const config: NextConfig = {
  reactStrictMode: true,
  // The BFF is a separate origin in development; rewriting keeps the browser
  // on one origin so the session cookie is first-party and no CORS preflight
  // sits in front of every checkout request.
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: `${process.env.BFF_URL ?? 'http://localhost:8080'}/:path*`,
      },
    ];
  },
};

export default config;

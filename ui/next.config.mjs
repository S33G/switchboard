/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  allowedDevOrigins: ['10.42.0.101'],

  // Bundle as static assets (served by nginx / Go file server).
  output: 'export',
  trailingSlash: true,
};

export default nextConfig;

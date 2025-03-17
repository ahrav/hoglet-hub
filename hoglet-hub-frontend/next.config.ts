/** @type {import('next').NextConfig} */
const nextConfig: import('next').NextConfig = {
  output: "standalone",
  // Ensure Next.js serves static assets correctly in Kubernetes
  // TODO: Come back to this when things get real.
  assetPrefix: process.env.NODE_ENV === "production" ? "" : undefined,
  // Disable ESLint during production builds
  eslint: {
    // Only run ESLint during development
    ignoreDuringBuilds: true,
  },
};

export default nextConfig;

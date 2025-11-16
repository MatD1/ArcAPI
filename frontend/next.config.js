/** @type {import('next').NextConfig} */
// Load env vars from parent directory .env file if they exist
const fs = require('fs');
const path = require('path');

// Try to load from parent .env file
const parentEnvPath = path.join(__dirname, '..', '.env');
if (fs.existsSync(parentEnvPath)) {
  const envFile = fs.readFileSync(parentEnvPath, 'utf8');
  envFile.split('\n').forEach((line) => {
    // Skip comments and empty lines
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) return;
    
    // Match key=value, handling quoted values
    const match = trimmed.match(/^([^=:#\s]+)\s*=\s*(.*)$/);
    if (match) {
      const key = match[1].trim();
      let value = match[2].trim();
      // Remove surrounding quotes (single or double)
      if ((value.startsWith('"') && value.endsWith('"')) || 
          (value.startsWith("'") && value.endsWith("'"))) {
        value = value.slice(1, -1);
      }
      if (key.startsWith('NEXT_PUBLIC_') && !process.env[key]) {
        process.env[key] = value;
      }
    }
  });
}

const nextConfig = {
  reactStrictMode: true,
  output: 'export',
  distDir: 'out',
  trailingSlash: true,
  images: {
    unoptimized: true,
  },
  env: {
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || '',
    NEXTAUTH_URL: process.env.NEXTAUTH_URL || '',
    NEXTAUTH_SECRET: process.env.NEXTAUTH_SECRET || '',
    GITHUB_CLIENT_ID: process.env.GITHUB_CLIENT_ID || '',
    GITHUB_CLIENT_SECRET: process.env.GITHUB_CLIENT_SECRET || '',
    NEXT_PUBLIC_APPWRITE_ENABLED: process.env.NEXT_PUBLIC_APPWRITE_ENABLED || 'false',
    NEXT_PUBLIC_APPWRITE_ENDPOINT: process.env.NEXT_PUBLIC_APPWRITE_ENDPOINT || '',
    NEXT_PUBLIC_APPWRITE_PROJECT_ID: process.env.NEXT_PUBLIC_APPWRITE_PROJECT_ID || '',
    NEXT_PUBLIC_APPWRITE_DATABASE_ID: process.env.NEXT_PUBLIC_APPWRITE_DATABASE_ID || 'arcapi',
  },
}

module.exports = nextConfig


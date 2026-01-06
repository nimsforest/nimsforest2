/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './pages/**/*.{js,ts,jsx,tsx,mdx}',
    './components/**/*.{js,ts,jsx,tsx,mdx}',
    './app/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        forest: {
          dark: '#1a1a2e',
          medium: '#16213e',
          light: '#0f3460',
          accent: '#4ade80',
        },
      },
    },
  },
  plugins: [],
};

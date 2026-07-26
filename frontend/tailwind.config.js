/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      colors: {
        bg: '#0f1220',
        surface: '#171a2b',
        surface2: '#1e2237',
        line: '#2a2f4a',
        brand: '#5b8cff',
        brand2: '#7c5cff',
        ok: '#2fbf71',
        danger: '#ff5d6c',
        warn: '#ffb020',
      },
    },
  },
  plugins: [],
}

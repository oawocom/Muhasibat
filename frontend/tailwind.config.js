/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      colors: {
        // Theme-aware (driven by CSS variables — see index.css).
        bg: 'var(--bg)',
        surface: 'var(--surface)',
        surface2: 'var(--surface2)',
        line: 'var(--line)',
        text: 'var(--text)',
        muted: 'var(--muted)',
        // Fixed accents.
        brand: '#5b8cff',
        brand2: '#7c5cff',
        ok: '#16a34a',
        danger: '#e5484d',
        warn: '#d97706',
      },
      boxShadow: {
        soft: '0 1px 2px rgba(16,24,40,.04), 0 8px 24px rgba(16,24,40,.06)',
      },
    },
  },
  plugins: [],
}

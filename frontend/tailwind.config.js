/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        gray: {
          950: '#080b12',
          925: '#0c1018',
          900: '#111827',
        },
        cyan: {
          450: '#22d3ee',
        },
        brand: {
          DEFAULT: '#06b6d4',
          dim: '#0891b2',
          glow: 'rgba(6,182,212,0.25)',
        },
      },
      fontFamily: {
        mono: ['JetBrains Mono', 'Fira Code', 'ui-monospace', 'monospace'],
      },
      backgroundImage: {
        'grid-pattern': "url(\"data:image/svg+xml,%3Csvg width='40' height='40' viewBox='0 0 40 40' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%2306b6d4' fill-opacity='0.04'%3E%3Cpath d='M0 40L40 0H20L0 20M40 40V20L20 40'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E\")",
        'dot-pattern': "radial-gradient(circle, rgba(6,182,212,0.08) 1px, transparent 1px)",
      },
      backgroundSize: {
        'dot-sm': '24px 24px',
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'glow-pulse': 'glowPulse 2s ease-in-out infinite',
        'slide-in': 'slideIn 0.2s ease-out',
        'fade-in': 'fadeIn 0.3s ease-out',
      },
      keyframes: {
        glowPulse: {
          '0%, 100%': { opacity: '1', boxShadow: '0 0 6px 1px rgba(6,182,212,0.5)' },
          '50%': { opacity: '0.7', boxShadow: '0 0 12px 3px rgba(6,182,212,0.8)' },
        },
        slideIn: {
          from: { opacity: '0', transform: 'translateY(-4px)' },
          to: { opacity: '1', transform: 'translateY(0)' },
        },
        fadeIn: {
          from: { opacity: '0' },
          to: { opacity: '1' },
        },
      },
      boxShadow: {
        'glow-cyan': '0 0 20px rgba(6,182,212,0.3)',
        'glow-cyan-sm': '0 0 8px rgba(6,182,212,0.4)',
        'glow-red': '0 0 12px rgba(239,68,68,0.4)',
        'glow-green': '0 0 12px rgba(52,211,153,0.4)',
        'inner-glow': 'inset 0 1px 0 rgba(255,255,255,0.05)',
      },
    },
  },
  plugins: [],
}

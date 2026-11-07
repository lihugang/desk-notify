/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./card.html",
    "./settings.html",
    "./history.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: [
          '"CMU Sans Serif"',
          '"Noto Sans SC"',
          '"Noto Sans"',
          '"Noto Sans Serif"',
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          '"Segoe UI"',
          'Roboto',
          '"Helvetica Neue"',
          '"PingFang SC"',
          '"Hiragino Sans GB"',
          '"Microsoft YaHei"',
          '"WenQuanYi Micro Hei"',
          'sans-serif',
        ],
      },
      colors: {
        primary: {
          50: '#eff6ff',
          100: '#dbeafe',
          500: '#3b82f6',
          600: '#2563eb',
          700: '#1d4ed8',
        }
      }
    },
  },
  plugins: [],
}

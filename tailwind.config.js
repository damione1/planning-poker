/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./web/templates/**/*.templ",
    "./web/templates/**/*.html",
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#f5f7fa',
          100: '#eaeef4',
          200: '#d0dae7',
          300: '#a8bbd4',
          400: '#7997bd',
          500: '#5779a7',
          600: '#43608c',
          700: '#374e72',
          800: '#30425f',
          900: '#2c3a50',
        },
      },
    },
  },
  plugins: [],
}

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./assets/server/views/**/*.{html,js}"],
  theme: {
    extend: {
      fontFamily: {
        Kanit: ["Kanit, sans-serif"],
      },
    },
  },
  plugins: [require("daisyui")],
  daisyui: {
    themes: ["dark"],
  },
};

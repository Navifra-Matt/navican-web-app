import type { Config } from 'tailwindcss'
import daisyui from 'daisyui'

export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  plugins: [
    require('daisyui'),
  ],
  daisyui: {
    themes: [
      {
        business: {
          ...require("daisyui/src/theming/themes")["business"],
          "primary": "#3b82f6", // Bright Blue for primary actions
          "primary-content": "#ffffff",
          "secondary": "#1e40af", // Darker Blue for secondary
          "accent": "#0ea5e9", // Sky Blue for accents
          "neutral": "#1e293b", // Slate 800
          "base-100": "#0f172a", // Slate 900 (Very dark blue-grey background)
          "base-200": "#1e293b", // Slate 800 (Lighter blue-grey)
          "base-300": "#334155", // Slate 700
          "base-content": "#f8fafc", // Slate 50
        },
      },
      "dark",
      "light"
    ],
  },
} satisfies Config

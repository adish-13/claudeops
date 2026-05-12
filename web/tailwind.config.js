/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        bg: "#0b0d10",
        panel: "#14171c",
        panel2: "#0f1216",
        hover: "#1a1e25",
        border: "#232831",
        "border-soft": "#1a1e26",
        fg: "#e6edf3",
        muted: "#8b949e",
        dim: "#6b7280",
        accent: "#58a6ff",
        warm: "#f0883e",
        ok: "#3fb950",
        err: "#f85149",
        add: "#56d364",
        del: "#f85149",
      },
      fontFamily: {
        sans: ["-apple-system", "BlinkMacSystemFont", '"SF Pro Text"', '"Segoe UI"', "Helvetica", "Arial", "sans-serif"],
        mono: ["ui-monospace", "SFMono-Regular", "Menlo", "monospace"],
      },
    },
  },
  plugins: [],
};

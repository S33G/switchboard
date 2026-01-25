import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}"],
  theme: {
    extend: {
      boxShadow: {
        soft: "0 18px 40px rgba(15, 23, 42, 0.12)",
      },
    },
  },
  plugins: [],
};

export default config;

import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}"],
  theme: {
    extend: {
      boxShadow: {
        soft: "0 18px 40px rgba(15, 23, 42, 0.12)",
      },
      keyframes: {
        "container-enter": {
          "0%": {
            opacity: "0",
            transform: "scale(0.95) translateY(-8px)",
          },
          "100%": {
            opacity: "1",
            transform: "scale(1) translateY(0)",
          },
        },
        "container-pulse": {
          "0%, 100%": {
            boxShadow: "0 0 0 0 rgba(56, 189, 248, 0)",
          },
          "50%": {
            boxShadow: "0 0 0 4px rgba(56, 189, 248, 0.3)",
          },
        },
        "row-enter": {
          "0%": {
            opacity: "0",
            transform: "translateX(-12px)",
          },
          "100%": {
            opacity: "1",
            transform: "translateX(0)",
          },
        },
        "row-pulse": {
          "0%, 100%": {
            backgroundColor: "transparent",
          },
          "50%": {
            backgroundColor: "rgba(56, 189, 248, 0.1)",
          },
        },
      },
      animation: {
        "container-enter": "container-enter 0.4s ease-out forwards",
        "container-pulse": "container-pulse 0.6s ease-in-out",
        "row-enter": "row-enter 0.3s ease-out forwards",
        "row-pulse": "row-pulse 0.6s ease-in-out",
      },
    },
  },
  plugins: [],
};

export default config;

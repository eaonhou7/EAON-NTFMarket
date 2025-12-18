import type { Config } from "tailwindcss";

export default {
  content: [
    "./pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./components/**/*.{js,ts,jsx,tsx,mdx}",
    "./app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      fontFamily: {
        'poppins': ['Poppins', 'sans-serif'],
      },
      colors: {
        background: "var(--background)",
        foreground: "var(--foreground)",
        primary: "var(--primary)",
        // 主题色系统 - 基于 #B8F703 和 #2C2E37
        // 引用 globals.css 中定义的 CSS 变量
        theme: {
          // 主色 - 亮绿色系
          primary: {
            DEFAULT: "var(--theme-primary)",
            light: "var(--theme-primary-light)",
            lighter: "var(--theme-primary-lighter)",
            dark: "var(--theme-primary-dark)",
            darker: "var(--theme-primary-darker)",
            50: "var(--theme-primary-50)",
            100: "var(--theme-primary-100)",
            200: "var(--theme-primary-200)",
            300: "var(--theme-primary-300)",
            400: "var(--theme-primary-400)",
            500: "var(--theme-primary-500)",
            600: "var(--theme-primary-600)",
            700: "var(--theme-primary-700)",
            800: "var(--theme-primary-800)",
            900: "var(--theme-primary-900)",
          },
          // 深色系
          dark: {
            DEFAULT: "var(--theme-dark)",
            light: "var(--theme-dark-light)",
            lighter: "var(--theme-dark-lighter)",
            dark: "var(--theme-dark-dark)",
            darker: "var(--theme-dark-darker)",
            50: "var(--theme-dark-50)",
            100: "var(--theme-dark-100)",
            200: "var(--theme-dark-200)",
            300: "var(--theme-dark-300)",
            400: "var(--theme-dark-400)",
            500: "var(--theme-dark-500)",
            600: "var(--theme-dark-600)",
            700: "var(--theme-dark-700)",
            800: "var(--theme-dark-800)",
            900: "var(--theme-dark-900)",
          },
          // 辅助色
          accent: {
            DEFAULT: "var(--theme-accent)",
            muted: "var(--theme-accent-muted)",
          },
          // 中性色（这些可以保持静态值，因为通常不需要动态切换）
          neutral: {
            white: "#FFFFFF",
            gray: {
              50: "#FAFAFA",
              100: "#F5F5F5",
              200: "#E5E5E5",
              300: "#D4D4D4",
              400: "#A3A3A3",
              500: "#737373",
              600: "#525252",
              700: "#404040",
              800: "#262626",
              900: "#171717",
            },
          },
        },
      },
    },
  },
  plugins: [],
} satisfies Config;

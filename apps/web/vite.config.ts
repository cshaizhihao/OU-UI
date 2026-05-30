import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  base: "./",
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          echarts: ["echarts/core", "echarts/components", "echarts/charts", "echarts/renderers"]
        }
      }
    }
  },
  server: {
    port: 5173,
    proxy: {
      "/ou-ui/api": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true
      }
    }
  }
});

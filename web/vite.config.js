import { defineConfig } from "vite";

const hmrHost = process.env.VITE_HMR_HOST || "localhost";
const publicBase =
  process.env.PUBLIC_BASE_URL ||
  `http://localhost:${process.env.PORT || 3000}`;

export default defineConfig({
  server: {
    // Tailscale / LAN: accept non-loopback; HMR host follows PUBLIC_BASE_URL.
    host: true,
    cors: {
      origin: [publicBase, `http://localhost:${process.env.PORT || 3000}`]
    },
    hmr: {
      host: hmrHost
    }
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
    rollupOptions: {
      input: "src/main.js",
      output: {
        entryFileNames: "assets/app.js",
        assetFileNames: "assets/[name][extname]"
      }
    }
  }
});

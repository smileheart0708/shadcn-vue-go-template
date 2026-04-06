import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig, loadEnv } from 'vite'
import { compression, defineAlgorithm } from 'vite-plugin-compression2'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')

  return {
    define: {
      __VITE_API_MOCKING__: env.VITE_API_MOCKING === 'true',
    },
    plugins: [
      tailwindcss(),
      vue(),
      compression({
        include: [/\.(css|html|js|json|map|svg|txt|xml)$/],
        algorithms: [defineAlgorithm('gzip', { level: 9 })],
        skipIfLargerOrEqual: true,
      }),
    ],
    resolve: { tsconfigPaths: true },
    server: { proxy: { '/api': { target: 'http://localhost:8080' } } },
    build: { target: 'es2022' },
  }
})

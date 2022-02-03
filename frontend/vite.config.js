import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
    plugins: [vue()],
    server: {
        port: 80,
        proxy: {
            '/api': 'http://localhost:3000',
            '/oauth': 'http://localhost:3000',
        }
    }
})

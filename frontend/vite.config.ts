import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

// 说明:
// 原配置使用 target: 'http://backend:5001' 在本地运行 (npm run dev) 时会解析不到 backend 主机，导致 /api 请求失败 (ENOTFOUND backend)
// 后端 docker-compose.golang.yml 暴露端口为 5004，因此默认改为 http://localhost:5004
// 可通过环境变量 VITE_DEV_API_TARGET 覆盖 (例如指向容器网络别名)

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const devTarget = env.VITE_DEV_API_TARGET || 'http://localhost:5004'
  return {
    plugins: [react()],
    server: {
      proxy: {
        '/api': {
          target: devTarget,
          changeOrigin: true,
          secure: false,
        }
      }
    },
    build: {
      chunkSizeWarningLimit: 700, // 放宽阈值
      rollupOptions: {
        output: {
          manualChunks(id) {
            if (id.includes('node_modules')) {
              if (id.includes('react-router') || id.includes('@remix-run')) return 'router';
              if (id.includes('react')) return 'react';
              if (id.includes('date-fns') || id.includes('dayjs') || id.includes('moment')) return 'date';
              if (id.includes('chart.js') || id.includes('recharts') ) return 'charts';
              if (id.includes('lodash')) return 'lodash';
            }
            return undefined;
          }
        }
      }
    }
  }
})
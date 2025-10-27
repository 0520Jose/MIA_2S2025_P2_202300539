import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': 'http://ec2-18-223-185-41.us-east-2.compute.amazonaws.com:8000'
    }
  }
});
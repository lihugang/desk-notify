import { createApp } from 'vue';
import TeacherApp from './teacher/TeacherApp.vue';
import './style.css';
import { registerSW } from 'virtual:pwa-register';
import { initPangu } from './utils/pangu';

// Initialize Pangu (盘古之白) text auto-spacing
initPangu();

// Register PWA service worker with auto-update
const updateSW = registerSW({
  onNeedRefresh() {
    if (confirm('发现新版本，是否立即更新？')) {
      updateSW(true);
    }
  },
  onOfflineReady() {
    console.log('DeskNotify 离线准备就绪');
  },
});

createApp(TeacherApp).mount('#app');

import { createApp } from 'vue';
import HistoryApp from './HistoryApp.vue';
import '../style.css';
import { initPangu } from '../utils/pangu';

// Initialize Pangu (盘古之白) text auto-spacing
initPangu();

createApp(HistoryApp).mount('#history-app');

import { createApp } from 'vue';
import SettingsApp from './SettingsApp.vue';
import '../style.css';
import { initPangu } from '../utils/pangu';

// Initialize Pangu (盘古之白) text auto-spacing
initPangu();

createApp(SettingsApp).mount('#settings-app');

import { createApp } from 'vue';
import CardApp from './CardApp.vue';
import '../style.css';
import { initPangu } from '../utils/pangu';

// Initialize Pangu (盘古之白) text auto-spacing
initPangu();

createApp(CardApp).mount('#card-app');

<template>
  <div class="h-screen bg-slate-100 text-slate-800 p-4 sm:p-6 flex flex-col">
    <div class="max-w-3xl mx-auto w-full bg-white rounded-2xl shadow-sm border border-slate-200 overflow-hidden flex flex-col flex-1 min-h-0">
      <!-- Main Content Area: Infinite Scroll History Feed -->
      <div class="p-5 sm:p-6 flex-1 overflow-y-auto">
        <HistoryList ref="historyListRef" title="班级全部历史通知" :page-size="10" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import HistoryList from '../components/HistoryList.vue';
import type { NotificationMessage } from '../types';

const historyListRef = ref<InstanceType<typeof HistoryList> | null>(null);
let ws: WebSocket | null = null;

onMounted(() => {
  connectWebSocket();

  window.addEventListener('focus', onWindowFocus);
  document.addEventListener('visibilitychange', onVisibilityChange);
  window.addEventListener('keydown', onKeyDown);
});

onUnmounted(() => {
  window.removeEventListener('focus', onWindowFocus);
  document.removeEventListener('visibilitychange', onVisibilityChange);
  window.removeEventListener('keydown', onKeyDown);
  if (ws) {
    ws.close();
  }
});

function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    closeWindow();
  }
}

function onWindowFocus() {
  historyListRef.value?.reload();
}

function onVisibilityChange() {
  if (document.visibilityState === 'visible') {
    historyListRef.value?.reload();
  }
}

function connectWebSocket() {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  ws = new WebSocket(`${protocol}//${window.location.host}/ws`);

  ws.onmessage = (event) => {
    try {
      const msg: NotificationMessage = JSON.parse(event.data);
      historyListRef.value?.prependItem(msg);
    } catch (e) {
      console.error('解析推送消息失败', e);
    }
  };

  ws.onclose = () => {
    setTimeout(connectWebSocket, 4000);
  };
}

function closeWindow() {
  if ((window as any).runtime?.WindowHide) {
    (window as any).runtime.WindowHide();
  }
  fetch('/api/window/history-hide', { method: 'POST' }).catch(() => {});
}
</script>

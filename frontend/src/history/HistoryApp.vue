<template>
  <div class="min-h-screen bg-slate-100 text-slate-800 p-4 sm:p-6 flex flex-col justify-between">
    <div class="max-w-3xl mx-auto w-full bg-white rounded-2xl shadow-sm border border-slate-200 overflow-hidden flex flex-col flex-1">
      <!-- Top Bar -->
      <div class="px-5 py-3.5 bg-slate-50 border-b border-slate-200 flex items-center justify-between select-none">
        <div class="flex items-center gap-2.5">
          <div class="w-8 h-8 rounded-xl bg-blue-600 text-white flex items-center justify-center shadow-xs">
            <History class="w-4 h-4" />
          </div>
          <div>
            <div class="flex items-center gap-2">
              <span class="text-sm font-bold text-slate-900">班级通知历史</span>
              <span class="px-2 py-0.5 rounded-full bg-blue-100 text-blue-700 text-xs font-semibold">
                {{ className || '班级电脑' }}
              </span>
            </div>
            <p class="text-[11px] text-slate-400">大屏接收的全部通知与课件存档</p>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <!-- Open Archive Dir -->
          <button
            @click="openArchiveDir"
            class="px-2.5 py-1.5 text-xs text-slate-600 hover:text-emerald-600 hover:bg-emerald-50 border border-slate-200 rounded-lg transition flex items-center gap-1.5 cursor-pointer"
            title="在资源管理器中打开我的文档/班级通知"
          >
            <FolderOpen class="w-3.5 h-3.5" />
            <span class="hidden sm:inline">打开存档目录</span>
          </button>

          <!-- Close Window -->
          <button
            @click="closeWindow"
            class="px-2.5 py-1.5 text-xs text-slate-500 hover:text-slate-800 hover:bg-slate-200/80 rounded-lg transition flex items-center gap-1 cursor-pointer"
            title="关闭窗口"
          >
            <X class="w-4 h-4" />
            <span class="hidden sm:inline">关闭</span>
          </button>
        </div>
      </div>

      <!-- Main Content Area: Infinite Scroll History Feed -->
      <div class="p-5 sm:p-6 flex-1 overflow-y-auto max-h-[calc(100vh-140px)]">
        <HistoryList ref="historyListRef" title="班级全部历史通知" :page-size="10" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import { History, FolderOpen, X } from 'lucide-vue-next';
import HistoryList from '../components/HistoryList.vue';
import type { NotificationMessage } from '../types';

const className = ref('');
const historyListRef = ref<InstanceType<typeof HistoryList> | null>(null);
let ws: WebSocket | null = null;

onMounted(async () => {
  await fetchConfig();
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

async function fetchConfig() {
  try {
    const res = await fetch('/api/config');
    if (res.ok) {
      const data = await res.json();
      if (data.className) {
        className.value = data.className;
      }
    }
  } catch (e) {
    console.error('获取配置失败', e);
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

async function openArchiveDir() {
  try {
    await fetch('/api/archive/open', { method: 'POST' });
  } catch (e) {
    console.error('打开存档目录失败', e);
  }
}

function closeWindow() {
  if ((window as any).runtime?.WindowHide) {
    (window as any).runtime.WindowHide();
  }
  fetch('/api/window/history-hide', { method: 'POST' }).catch(() => {});
}
</script>

<template>
  <div
    v-if="currentMessage"
    class="fixed inset-0 p-6 flex flex-col justify-start items-center bg-transparent pointer-events-auto"
  >
    <!-- Card Container -->
    <div
      class="w-full max-w-2xl bg-white/95 backdrop-blur-md rounded-3xl shadow-2xl border border-slate-200/80 overflow-hidden transform transition-all duration-300 animate-slide-in text-slate-800"
      :class="getCardBorderClass(currentMessage.level)"
    >
      <!-- Top Bar: Level, Time, Countdown & Close -->
      <div
        class="px-6 py-4 flex items-center justify-between border-b border-slate-100 select-none cursor-move"
        :class="getHeaderBgClass(currentMessage.level)"
        style="--wails-draggable: drag;"
      >
        <div class="flex items-center space-x-3 pointer-events-none">
          <span class="px-3 py-1 rounded-full text-xs font-bold tracking-wide uppercase shadow-xs" :class="getBadgeClass(currentMessage.level)">
            {{ getLevelLabel(currentMessage.level) }}
          </span>
          <span class="text-xs text-slate-500 font-medium">{{ classroomName }} · 桌面通知</span>
        </div>

        <div class="flex items-center space-x-3" style="--wails-draggable: no-drag;">
          <!-- Countdown Timer -->
          <div v-if="remainingSeconds > 0" class="flex items-center space-x-1.5 text-xs text-slate-500 bg-white/80 px-2.5 py-1 rounded-full border border-slate-200/60">
            <Clock class="w-3.5 h-3.5 text-slate-400" />
            <span class="font-mono font-semibold">{{ remainingSeconds }}s 后自动关闭</span>
          </div>

          <!-- Manual Close Button -->
          <button
            @click="closeCard"
            class="p-1.5 rounded-full hover:bg-slate-200/80 text-slate-400 hover:text-slate-700 transition cursor-pointer"
            title="关闭通知"
          >
            <X class="w-5 h-5" />
          </button>
        </div>
      </div>

      <!-- Main Body -->
      <div class="p-6 max-h-[70vh] overflow-y-auto space-y-5">
        <!-- Rendered Markdown Content -->
        <div
          class="prose prose-slate max-w-none text-lg leading-relaxed font-normal"
          v-html="renderedMarkdown"
        ></div>

        <!-- Inline Images -->
        <div v-if="currentMessage.images?.length" class="grid grid-cols-2 gap-3 pt-2">
          <div
            v-for="(imgUrl, idx) in currentMessage.images"
            :key="idx"
            class="rounded-xl overflow-hidden border border-slate-200 bg-slate-50 shadow-xs cursor-pointer hover:opacity-95 transition"
            @click="openImage(imgUrl)"
          >
            <img :src="imgUrl" class="w-full max-h-72 object-contain" />
          </div>
        </div>

        <!-- Attachments Cards -->
        <div v-if="currentMessage.attachments?.length" class="pt-3 border-t border-slate-100 space-y-2">
          <p class="text-xs font-bold text-slate-400 uppercase tracking-wider">课件与附件</p>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
            <a
              v-for="att in currentMessage.attachments"
              :key="att.id"
              :href="att.url"
              target="_blank"
              download
              class="flex items-center p-3 rounded-2xl bg-slate-50 hover:bg-blue-50/60 border border-slate-200 hover:border-blue-200 transition group"
            >
              <div class="w-9 h-9 rounded-xl bg-blue-100 text-blue-600 flex items-center justify-center mr-3 flex-shrink-0 group-hover:scale-105 transition-transform">
                <FileText class="w-5 h-5" />
              </div>
              <div class="truncate flex-1">
                <p class="text-sm font-semibold text-slate-800 truncate group-hover:text-blue-600 transition">{{ att.name }}</p>
                <p class="text-xs text-slate-400">{{ formatSize(att.size) }}</p>
              </div>
              <Download class="w-4 h-4 text-slate-400 group-hover:text-blue-600 ml-2 flex-shrink-0" />
            </a>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { Clock, X, FileText, Download } from 'lucide-vue-next';
import { marked } from 'marked';
import type { NotificationMessage } from '../types';

const currentMessage = ref<NotificationMessage | null>(null);
const classroomName = ref('班级大屏');
const remainingSeconds = ref(0);
let timer: any = null;

const renderedMarkdown = computed(() => {
  if (!currentMessage.value?.content) return '';
  return marked.parse(currentMessage.value.content);
});

onMounted(() => {
  connectWebSocket();
});

function connectWebSocket() {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const ws = new WebSocket(`${protocol}//${window.location.host}/ws`);

  ws.onmessage = (event) => {
    try {
      const msg: NotificationMessage = JSON.parse(event.data);
      showMessage(msg);
    } catch (e) {
      console.error('解析推送消息失败', e);
    }
  };

  ws.onclose = () => {
    setTimeout(connectWebSocket, 3000);
  };
}

function showMessage(msg: NotificationMessage) {
  currentMessage.value = msg;
  if (msg.soundEnabled !== false) {
    playAlertSound(msg.soundUrl);
  }

  if (timer) clearInterval(timer);
  if (msg.duration > 0) {
    remainingSeconds.value = msg.duration;
    timer = setInterval(() => {
      remainingSeconds.value--;
      if (remainingSeconds.value <= 0) {
        closeCard();
      }
    }, 1000);
  } else {
    remainingSeconds.value = 0;
  }
}

function closeCard() {
  if (timer) clearInterval(timer);
  currentMessage.value = null;
  // If running in Wails runtime, notify backend to hide window
  if ((window as any).runtime?.WindowHide) {
    (window as any).runtime.WindowHide();
  }
  fetch('/api/window/hide', { method: 'POST' }).catch(() => {});
}

function playAlertSound(soundUrl?: string) {
  try {
    const url = soundUrl || `/api/sound/current?t=${Date.now()}`;
    const audio = new Audio(url);
    audio.play().catch((err) => {
      console.warn('播放提示音失败:', err);
    });
  } catch (e) {}
}

function openImage(url: string) {
  window.open(url, '_blank');
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
}

function getLevelLabel(level: string): string {
  switch (level) {
    case 'urgent': return '紧急通知';
    case 'important': return '重要通知';
    default: return '班级通知';
  }
}

function getBadgeClass(level: string): string {
  switch (level) {
    case 'urgent': return 'bg-red-500 text-white';
    case 'important': return 'bg-amber-500 text-white';
    default: return 'bg-blue-600 text-white';
  }
}

function getHeaderBgClass(level: string): string {
  switch (level) {
    case 'urgent': return 'bg-red-50/50';
    case 'important': return 'bg-amber-50/50';
    default: return 'bg-blue-50/30';
  }
}

function getCardBorderClass(level: string): string {
  switch (level) {
    case 'urgent': return 'ring-2 ring-red-500/30 shadow-red-500/10';
    case 'important': return 'ring-2 ring-amber-500/30 shadow-amber-500/10';
    default: return 'ring-1 ring-blue-500/20 shadow-blue-500/5';
  }
}
</script>

<style scoped>
@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(-20px) scale(0.97);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}
.animate-slide-in {
  animation: slideIn 0.3s cubic-bezier(0.16, 1, 0.3, 1) forwards;
}
</style>

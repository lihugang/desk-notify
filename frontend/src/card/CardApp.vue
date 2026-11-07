<template>
  <div
    v-if="currentMessage || viewMode === 'history'"
    class="fixed inset-0 p-4 sm:p-6 flex flex-col justify-start items-center bg-transparent pointer-events-auto"
  >
    <!-- Card Container -->
    <div
      class="w-full max-w-2xl bg-white/95 backdrop-blur-md rounded-3xl shadow-2xl border border-slate-200/80 overflow-hidden transform transition-all duration-300 animate-slide-in text-slate-800 flex flex-col"
      :class="currentMessage ? getCardBorderClass(currentMessage.level) : 'ring-1 ring-blue-500/20'"
    >
      <!-- Top Bar: Level, Time, Countdown, History Toggle & Close -->
      <div
        class="px-6 py-3.5 flex items-center justify-between border-b border-slate-100 select-none cursor-move flex-shrink-0"
        :class="currentMessage ? getHeaderBgClass(currentMessage.level) : 'bg-slate-50'"
        style="--wails-draggable: drag;"
      >
        <div class="flex items-center space-x-3 pointer-events-none">
          <span
            v-if="viewMode === 'current' && currentMessage"
            class="px-3 py-1 rounded-full text-xs font-bold tracking-wide uppercase shadow-xs"
            :class="getBadgeClass(currentMessage.level)"
          >
            {{ getLevelLabel(currentMessage.level) }}
          </span>
          <span
            v-else
            class="px-3 py-1 rounded-full text-xs font-bold tracking-wide uppercase shadow-xs bg-indigo-600 text-white"
          >
            历史记录
          </span>
          <span class="text-xs text-slate-500 font-medium">{{ classroomName }} · 桌面通知</span>
        </div>

        <div class="flex items-center space-x-2" style="--wails-draggable: no-drag;">
          <!-- Countdown Timer (Current Mode only) -->
          <div
            v-if="viewMode === 'current' && remainingSeconds > 0"
            class="flex items-center space-x-1.5 text-xs text-slate-500 bg-white/80 px-2.5 py-1 rounded-full border border-slate-200/60"
          >
            <Clock class="w-3.5 h-3.5 text-slate-400" />
            <span class="font-mono font-semibold">{{ remainingSeconds }}s</span>
          </div>

          <!-- History Mode Toggle Button -->
          <button
            @click="toggleViewMode"
            class="px-2.5 py-1 rounded-lg text-xs font-medium border border-slate-200 transition flex items-center gap-1 cursor-pointer shadow-2xs"
            :class="viewMode === 'history' ? 'bg-blue-600 text-white border-blue-600' : 'bg-white hover:bg-slate-100 text-slate-600'"
            :title="viewMode === 'history' ? '返回当前通知' : '查看历史通知记录'"
          >
            <History class="w-3.5 h-3.5" />
            <span>{{ viewMode === 'history' ? '返回最新' : '历史通知' }}</span>
          </button>

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

      <!-- Body: History Feed View (Desktop Big Screen) -->
      <div v-if="viewMode === 'history'" class="p-5 max-h-[72vh] overflow-y-auto space-y-4">
        <HistoryList title="班级全部历史通知" :page-size="8" />
      </div>

      <!-- Body: Current Notification View -->
      <div v-else-if="currentMessage" class="p-6 max-h-[70vh] overflow-y-auto space-y-5">
        <!-- Rendered Markdown Content (No truncation) -->
        <div
          class="markdown-content text-slate-800 text-base sm:text-lg leading-relaxed font-normal"
          v-html="renderedMarkdown"
        ></div>

        <!-- Inline Images -->
        <div v-if="currentMessage.images?.length" class="grid grid-cols-2 gap-3 pt-2">
          <div
            v-for="(imgUrl, idx) in currentMessage.images"
            :key="idx"
            class="rounded-xl overflow-hidden border border-slate-200 bg-slate-50 shadow-xs cursor-pointer hover:opacity-95 transition"
            @click="activeLightboxImage = imgUrl"
          >
            <img :src="imgUrl" class="w-full max-h-72 object-contain" alt="通知图片" />
          </div>
        </div>

        <!-- Attachments Cards -->
        <div v-if="currentMessage.attachments?.length" class="pt-3 border-t border-slate-100 space-y-2">
          <p class="text-xs font-bold text-slate-400 uppercase tracking-wider">课件与附件 (点击直接下载)</p>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
            <a
              v-for="att in currentMessage.attachments"
              :key="att.id"
              :href="getDownloadUrl(att)"
              target="_blank"
              :download="att.name"
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

    <!-- Image Lightbox Modal -->
    <ImageLightbox :image-url="activeLightboxImage" @close="activeLightboxImage = null" />
  </div>

  <!-- Empty State Card (When no notification exists) -->
  <div
    v-else-if="showEmptyState"
    class="fixed inset-0 p-4 sm:p-6 flex flex-col justify-start items-center bg-transparent pointer-events-auto"
  >
    <div
      class="w-full max-w-md bg-white/95 backdrop-blur-md rounded-3xl shadow-2xl border border-slate-200/80 overflow-hidden transform transition-all duration-300 animate-slide-in text-slate-800 flex flex-col"
    >
      <!-- Top Bar -->
      <div
        class="px-6 py-3.5 flex items-center justify-between border-b border-slate-100 select-none bg-slate-50 cursor-move flex-shrink-0"
        style="--wails-draggable: drag;"
      >
        <div class="flex items-center space-x-2 pointer-events-none">
          <Bell class="w-4 h-4 text-blue-600" />
          <span class="text-xs text-slate-500 font-medium">{{ classroomName }} · 桌面通知</span>
        </div>

        <div class="flex items-center space-x-2" style="--wails-draggable: no-drag;">
          <!-- History Mode Toggle Button -->
          <button
            @click="viewMode = 'history'; showEmptyState = false"
            class="px-2.5 py-1 rounded-lg text-xs font-medium border border-slate-200 bg-white hover:bg-slate-100 text-slate-600 transition flex items-center gap-1 cursor-pointer shadow-2xs"
            title="查看历史通知记录"
          >
            <History class="w-3.5 h-3.5" />
            <span>历史通知</span>
          </button>

          <!-- Manual Close Button -->
          <button
            @click="closeCard"
            class="p-1.5 rounded-full hover:bg-slate-200/80 text-slate-400 hover:text-slate-700 transition cursor-pointer"
            title="关闭窗口"
          >
            <X class="w-5 h-5" />
          </button>
        </div>
      </div>

      <!-- Empty State Content -->
      <div class="p-8 text-center space-y-4">
        <div class="w-16 h-16 mx-auto rounded-2xl bg-slate-100/80 flex items-center justify-center text-slate-400">
          <Inbox class="w-8 h-8" />
        </div>
        <div>
          <h3 class="text-lg font-bold text-slate-800">暂无最新通知</h3>
          <p class="text-sm text-slate-400 mt-1">当前班级还没有收到任何教师通知</p>
        </div>
        <div class="pt-2">
          <button
            @click="closeCard"
            class="px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-xl text-sm font-medium transition cursor-pointer shadow-xs"
          >
            我知道了
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { Clock, X, FileText, Download, History, Bell, Inbox } from 'lucide-vue-next';
import { marked } from 'marked';
import type { NotificationMessage, Attachment } from '../types';
import HistoryList from '../components/HistoryList.vue';
import ImageLightbox from '../components/ImageLightbox.vue';

const currentMessage = ref<NotificationMessage | null>(null);
const classroomName = ref('班级大屏');
const remainingSeconds = ref(0);
const viewMode = ref<'current' | 'history'>('current');
const showEmptyState = ref(false);
const activeLightboxImage = ref<string | null>(null);
let timer: any = null;

const renderedMarkdown = computed(() => {
  if (!currentMessage.value?.content) return '';
  return marked.parse(currentMessage.value.content, { breaks: true, gfm: true }) as string;
});

onMounted(() => {
  connectWebSocket();
  fetchLatestNotification(false);

  window.addEventListener('focus', () => {
    if (!currentMessage.value && viewMode.value === 'current') {
      fetchLatestNotification(false);
    }
  });
});

function toggleViewMode() {
  if (viewMode.value === 'current') {
    viewMode.value = 'history';
    showEmptyState.value = false;
    if (timer) {
      clearInterval(timer);
      timer = null;
    }
  } else {
    viewMode.value = 'current';
    if (!currentMessage.value) {
      fetchLatestNotification(false);
    } else if (currentMessage.value.duration > 0 && remainingSeconds.value > 0) {
      startCountdown();
    }
  }
}

async function fetchLatestNotification(playSound: boolean = false) {
  try {
    const res = await fetch(`/api/notify/latest?t=${Date.now()}`);
    if (res.ok) {
      const data = await res.json();
      if (data.className) {
        classroomName.value = data.className;
      }
      if (data.hasNotification && data.notification) {
        showEmptyState.value = false;
        const msg: NotificationMessage = {
          ...data.notification,
          soundEnabled: data.soundEnabled,
          soundUrl: data.soundUrl
        };
        showMessage(msg, playSound);
      } else {
        currentMessage.value = null;
        if (viewMode.value !== 'history') {
          showEmptyState.value = true;
        }
      }
    }
  } catch (e) {
    console.error('获取最新通知失败', e);
  }
}

// Expose handlers to Wails / window runtime
(window as any).deskNotifyShowLatest = () => {
  viewMode.value = 'current';
  fetchLatestNotification(false);
};

(window as any).deskNotifyShowMessage = (msg: NotificationMessage) => {
  viewMode.value = 'current';
  showEmptyState.value = false;
  showMessage(msg, true);
};

function connectWebSocket() {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const ws = new WebSocket(`${protocol}//${window.location.host}/ws`);

  ws.onmessage = (event) => {
    try {
      const msg: NotificationMessage = JSON.parse(event.data);
      // When a new message comes in, switch back to current message view
      viewMode.value = 'current';
      showEmptyState.value = false;
      showMessage(msg, true);
    } catch (e) {
      console.error('解析推送消息失败', e);
    }
  };

  ws.onclose = () => {
    setTimeout(connectWebSocket, 3000);
  };
}

function showMessage(msg: NotificationMessage, playSound: boolean = true) {
  showEmptyState.value = false;
  currentMessage.value = msg;
  if (playSound && msg.soundEnabled !== false) {
    playAlertSound(msg.soundUrl);
  }

  if (timer) clearInterval(timer);
  if (msg.duration > 0) {
    remainingSeconds.value = msg.duration;
    startCountdown();
  } else {
    remainingSeconds.value = 0;
  }
}

function startCountdown() {
  if (timer) clearInterval(timer);
  timer = setInterval(() => {
    remainingSeconds.value--;
    if (remainingSeconds.value <= 0) {
      closeCard();
    }
  }, 1000);
}

function closeCard() {
  if (timer) clearInterval(timer);
  currentMessage.value = null;
  showEmptyState.value = false;
  viewMode.value = 'current';
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

function getDownloadUrl(att: Attachment): string {
  if (!att.url) return `/api/files/${att.id}?download=1`;
  return att.url.includes('?') ? `${att.url}&download=1` : `${att.url}?download=1`;
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

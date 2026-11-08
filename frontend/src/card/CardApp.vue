<template>
  <div
    class="fixed inset-0 p-3 sm:p-5 flex flex-col justify-start items-center bg-transparent pointer-events-auto cursor-default"
    @click.self="handleBackdropClick"
  >
    <!-- Card Container -->
    <div
      class="relative w-full max-w-2xl bg-white/95 backdrop-blur-md rounded-3xl shadow-2xl border border-slate-200/80 overflow-hidden transform animate-slide-in text-slate-800 flex flex-col h-full transition-[box-shadow,background-color] duration-300"
      :class="notifications.length > 0 && notifications[0].pinned ? 'ring-2 ring-amber-500/40 shadow-amber-500/10' : 'ring-1 ring-blue-500/20 shadow-blue-500/5'"
    >
      <!-- Top Resize Border (Vertical drag resize only) -->
      <div
        class="absolute top-0 left-0 right-0 h-2 cursor-ns-resize z-50 select-none hover:bg-blue-500/20 active:bg-blue-500/40 transition-colors"
        title="上下拖动调整窗口高度"
        data-no-drag
        style="-webkit-app-region: no-drag;"
        @mousedown.stop="handleResizeTop"
      ></div>

      <!-- Top Bar: Level, Time, Countdown, History Toggle, Minimize & Close -->
      <div
        class="px-6 py-3.5 flex items-center justify-between border-b border-slate-100 select-none cursor-move flex-shrink-0 drag-header"
        :class="notifications.length > 0 && notifications[0].pinned ? 'bg-amber-50/70' : 'bg-slate-50'"
        style="-webkit-app-region: drag; --wails-draggable: drag;"
        @mousedown="handleHeaderMouseDown"
      >
        <div class="flex items-center space-x-2.5 pointer-events-none">
          <span
            v-if="viewMode === 'current' && notifications.length > 0"
            class="px-2.5 py-1 rounded-full text-xs font-bold tracking-wide shadow-xs flex items-center gap-1"
            :class="notifications[0].pinned ? 'bg-amber-500 text-white' : 'bg-blue-600 text-white'"
          >
            <Bell class="w-3 h-3" />
            <span>最新通知</span>
          </span>
          <span
            v-else-if="viewMode === 'history'"
            class="px-2.5 py-1 rounded-full text-xs font-bold tracking-wide uppercase shadow-xs bg-indigo-600 text-white"
          >
            历史记录
          </span>
          <span
            v-else
            class="px-2.5 py-1 rounded-full text-xs font-bold tracking-wide shadow-xs flex items-center gap-1 bg-blue-600 text-white"
          >
            <Bell class="w-3 h-3" />
            <span>最新通知</span>
          </span>
          <span class="text-xs text-slate-500 font-medium">{{ classroomName }} · 桌面通知</span>
        </div>

        <div class="flex items-center space-x-2" style="-webkit-app-region: no-drag;" data-no-drag>
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
            :title="viewMode === 'history' ? '返回最新通知' : '查看历史通知记录'"
            data-no-drag
            style="-webkit-app-region: no-drag;"
          >
            <History class="w-3.5 h-3.5" />
            <span>{{ viewMode === 'history' ? '返回最新' : '历史通知' }}</span>
          </button>

          <!-- Minimize Button -->
          <button
            @click="minimizeCard"
            class="p-1.5 rounded-full hover:bg-slate-200/80 text-slate-400 hover:text-slate-700 transition cursor-pointer"
            title="最小化"
            data-no-drag
            style="-webkit-app-region: no-drag;"
          >
            <Minus class="w-4 h-4" />
          </button>

          <!-- Manual Close Button -->
          <button
            @click="closeCard"
            class="p-1.5 rounded-full hover:bg-slate-200/80 text-slate-400 hover:text-slate-700 transition cursor-pointer"
            title="关闭通知"
            data-no-drag
            style="-webkit-app-region: no-drag;"
          >
            <X class="w-5 h-5" />
          </button>
        </div>
      </div>

      <!-- Body: History Feed View (Desktop Big Screen 70% Frame) -->
      <div v-if="viewMode === 'history'" class="p-5 flex-1 overflow-y-auto space-y-4">
        <HistoryList title="班级全部历史通知" :page-size="8" />
      </div>

      <!-- Body: Current Notifications View (Last 90 Minutes) -->
      <div v-else-if="notifications.length > 0" class="p-5 overflow-y-auto flex-1 space-y-4">
        <div
          v-for="(item, idx) in notifications"
          :key="item.id || idx"
          class="bg-white/90 rounded-2xl p-4 sm:p-5 border border-slate-200/80 shadow-2xs space-y-3"
          :class="item.pinned ? 'ring-1 ring-amber-400/40 bg-amber-50/20' : ''"
        >
          <!-- Metadata Bar: Level & Time Point -->
          <div class="flex items-center justify-between text-xs pb-2 border-b border-slate-100/90 flex-wrap gap-2">
            <div class="flex items-center gap-2 flex-wrap">
              <span
                class="px-2.5 py-0.5 rounded-full font-semibold text-[11px] shadow-2xs flex items-center gap-1"
                :class="item.pinned ? 'bg-amber-100 text-amber-800 border border-amber-200' : 'bg-blue-50 text-blue-700 border border-blue-200'"
              >
                <Pin v-if="item.pinned" class="w-3 h-3 text-amber-600" />
                <span>{{ item.pinned ? '置顶提醒' : '班级通知' }}</span>
              </span>
            </div>

            <div class="flex items-center gap-2 flex-wrap">
              <div v-if="idx === 0 && remainingSeconds > 0" class="flex items-center gap-1 text-xs font-mono bg-blue-50 text-blue-700 border border-blue-200 px-2 py-0.5 rounded-full">
                <Clock class="w-3 h-3" />
                <span>{{ remainingSeconds }}s 后自动关闭</span>
              </div>

              <!-- Time Point Display aligned right -->
              <span class="text-slate-600 font-mono text-[11px] font-medium flex items-center gap-1 bg-slate-50 border border-slate-200/80 px-2.5 py-0.5 rounded-md">
                <Clock class="w-3 h-3 text-slate-400" />
                <span>{{ formatExactTime(item.createdAt) }}</span>
              </span>
            </div>
          </div>

          <!-- Markdown Content -->
          <div
            class="markdown-content text-slate-800 leading-relaxed font-normal"
            :style="{ fontSize: currentFontSize }"
            v-html="renderMarkdown(item.content)"
          ></div>

          <!-- Inline Images -->
          <div v-if="item.images && item.images.length > 0" class="grid grid-cols-2 gap-2.5 pt-1">
            <div
              v-for="(imgUrl, imgIdx) in item.images"
              :key="imgIdx"
              class="rounded-xl overflow-hidden border border-slate-200 bg-slate-50 shadow-xs cursor-pointer hover:opacity-95 transition"
              @click="activeLightboxImage = imgUrl"
            >
              <img :src="imgUrl" class="w-full max-h-60 object-contain" alt="通知图片" />
            </div>
          </div>

          <!-- Attachments Cards -->
          <div v-if="item.attachments && item.attachments.length > 0" class="pt-2 border-t border-slate-100 space-y-2">
            <p class="text-xs font-bold text-slate-400 uppercase tracking-wider">课件与附件</p>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
              <div
                v-for="att in item.attachments"
                :key="att.id"
                class="flex items-center justify-between p-2.5 rounded-xl bg-slate-50 hover:bg-blue-50/60 border border-slate-200 hover:border-blue-200 transition group select-none cursor-pointer"
                @click="openAttachment(att)"
                :title="'点击直接打开：' + att.name"
              >
                <div class="flex items-center space-x-2.5 truncate min-w-0 pr-2">
                  <div class="w-8 h-8 rounded-lg bg-blue-100 text-blue-600 flex items-center justify-center flex-shrink-0 group-hover:scale-105 transition-transform">
                    <FileText class="w-4 h-4" />
                  </div>
                  <div class="truncate">
                    <p class="text-xs font-semibold text-slate-800 truncate group-hover:text-blue-600 transition">{{ att.name }}</p>
                    <p class="text-[10px] text-slate-400">{{ formatSize(att.size) }}</p>
                  </div>
                </div>
                <div class="flex items-center gap-1 flex-shrink-0" @click.stop>
                  <button
                    type="button"
                    @click="revealAttachment(att)"
                    class="p-1.5 rounded-lg bg-white hover:bg-blue-600 text-slate-500 hover:text-white border border-slate-200 hover:border-blue-600 shadow-2xs transition cursor-pointer"
                    title="在文件夹中打开并选择此文件"
                  >
                    <Folder class="w-3.5 h-3.5" />
                  </button>
                  <button
                    type="button"
                    @click="openAttachment(att)"
                    class="p-1.5 rounded-lg bg-white hover:bg-blue-600 text-slate-500 hover:text-white border border-slate-200 hover:border-blue-600 shadow-2xs transition cursor-pointer"
                    title="直接打开此文件"
                  >
                    <ExternalLink class="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Body: Empty State View (When no notification exists) -->
      <div v-else class="p-8 flex-1 flex flex-col items-center justify-center text-center space-y-4">
        <div class="w-16 h-16 mx-auto rounded-2xl bg-slate-100/80 flex items-center justify-center text-slate-400">
          <Inbox class="w-8 h-8" />
        </div>
        <div>
          <h3 class="text-lg font-bold text-slate-800">暂无最新通知</h3>
          <p class="text-sm text-slate-400 mt-1">最近 90 分钟内暂无收到新的班级通知</p>
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

      <!-- Bottom Resize Border (Vertical drag resize only) -->
      <div
        class="absolute bottom-0 left-0 right-0 h-2.5 cursor-ns-resize z-50 select-none hover:bg-blue-500/20 active:bg-blue-500/40 transition-colors"
        title="上下拖动调整窗口高度"
        data-no-drag
        style="-webkit-app-region: no-drag;"
        @mousedown.stop="handleResizeBottom"
      ></div>
    </div>

    <!-- Image Lightbox Modal -->
    <ImageLightbox :image-url="activeLightboxImage" @close="activeLightboxImage = null" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue';
import { Clock, X, FileText, Folder, ExternalLink, History, Bell, Inbox, Minus, Pin } from 'lucide-vue-next';
import { marked } from 'marked';
import type { NotificationMessage, Attachment } from '../types';
import HistoryList from '../components/HistoryList.vue';
import ImageLightbox from '../components/ImageLightbox.vue';
import { spacePage } from '../utils/pangu';

const notifications = ref<NotificationMessage[]>([]);
const classroomName = ref('班级大屏');
const remainingSeconds = ref(0);
const viewMode = ref<'current' | 'history'>('current');
const activeLightboxImage = ref<string | null>(null);
const currentFontSize = ref(
  (typeof localStorage !== 'undefined' && localStorage.getItem('desknotify_font_size')) || '16px'
);
let timer: any = null;

// Mode-specific height constants and persistence
const DEFAULT_CURRENT_HEIGHT = 420; // 默认最新通知大小：顶部拖动条 + 2 条通知
const DEFAULT_HISTORY_HEIGHT = 760; // 默认历史通知大小：大屏长列表
let lastAppliedHeight = 0;
let resizeDebounceTimer: any = null;
let activeFetchId = 0;
let isFetchingLatest = false;

function getSavedHeightForMode(mode: 'current' | 'history'): number {
  try {
    const key = mode === 'current' ? 'desknotify_card_height_current' : 'desknotify_card_height_history';
    const val = localStorage.getItem(key);
    if (val) {
      const parsed = parseInt(val, 10);
      if (!isNaN(parsed) && parsed >= 240 && parsed <= 2400) {
        return parsed;
      }
    }
  } catch {}
  return mode === 'current' ? DEFAULT_CURRENT_HEIGHT : DEFAULT_HISTORY_HEIGHT;
}

async function syncWindowHeight(mode: 'current' | 'history') {
  const targetHeight = getSavedHeightForMode(mode);
  if (lastAppliedHeight === targetHeight) return;
  lastAppliedHeight = targetHeight;
  try {
    await fetch('/api/window/card-height', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ height: targetHeight }),
    });
  } catch (e) {
    console.warn('调整窗口高度失败:', e);
  }
}

function handleWindowResize() {
  if (resizeDebounceTimer) {
    clearTimeout(resizeDebounceTimer);
  }
  resizeDebounceTimer = setTimeout(() => {
    const currentHeight = window.innerHeight;
    if (currentHeight >= 240 && currentHeight <= 2400) {
      const diff = Math.abs(currentHeight - lastAppliedHeight);
      if (diff >= 5) {
        lastAppliedHeight = currentHeight;
        const key = viewMode.value === 'current' ? 'desknotify_card_height_current' : 'desknotify_card_height_history';
        try {
          localStorage.setItem(key, String(currentHeight));
        } catch {}
      }
    }
  }, 200);
}

function toggleViewMode() {
  if (viewMode.value === 'current') {
    viewMode.value = 'history';
    syncWindowHeight('history');
    if (timer) {
      clearInterval(timer);
      timer = null;
    }
  } else {
    viewMode.value = 'current';
    syncWindowHeight('current');
    if (notifications.value.length === 0) {
      fetchLatestNotification(false);
    } else if (notifications.value[0]?.duration > 0 && remainingSeconds.value > 0) {
      startCountdown();
    }
  }
}

function renderMarkdown(content?: string): string {
  if (!content) return '';
  const html = marked.parse(content, { breaks: true, gfm: true }) as string;
  nextTick(() => spacePage());
  return html;
}

function handleWindowInteraction() {
  fetch('/api/window/stop-flash', { method: 'POST' }).catch(() => {});
}

// Transparent backdrop click handler -> closes the card window
function handleBackdropClick() {
  closeCard();
}

function pruneExpiredNotifications() {
  const cutoff = Date.now() - 90 * 60 * 1000;
  notifications.value = notifications.value.filter((item) => {
    if (!item.createdAt) return true;
    const t = new Date(item.createdAt).getTime();
    return isNaN(t) || t >= cutoff;
  });
}

function handleIncomingNotification(msg: any) {
  viewMode.value = 'current';
  syncWindowHeight('current');

  // 1. If msg contains full 90-minute notifications list from backend WebSocket
  if (msg.notifications && Array.isArray(msg.notifications) && msg.notifications.length > 0) {
    notifications.value = msg.notifications.map((item: any) => ({
      ...item,
      soundEnabled: msg.soundEnabled,
      soundUrl: msg.soundUrl,
    }));
  } else {
    // 2. Prepend single notification and fetch recent 90-min list in background to sync
    const exists = notifications.value.some((n) => n.id === msg.id);
    if (!exists) {
      notifications.value = [msg, ...notifications.value];
    }
    fetchLatestNotification(false);
  }

  pruneExpiredNotifications();

  if (msg.soundEnabled !== false) {
    playAlertSound(msg.soundUrl);
  }

  if (msg.duration > 0) {
    remainingSeconds.value = msg.duration;
    startCountdown();
  } else {
    if (timer) {
      clearInterval(timer);
      timer = null;
    }
    remainingSeconds.value = 0;
  }
}

// Window header dragging
function handleHeaderMouseDown(e: MouseEvent) {
  if (e.button !== 0) return;
  const target = e.target as HTMLElement;
  if (target.closest('button, a, input, select, textarea, [data-no-drag]')) {
    return;
  }
  handleWindowInteraction();
  if ((window as any).chrome?.webview?.postMessage) {
    (window as any).chrome.webview.postMessage('wails:drag');
  } else if ((window as any).webkit?.messageHandlers?.['external']?.postMessage) {
    (window as any).webkit.messageHandlers['external'].postMessage('wails:drag');
  }
}

// Top and bottom border vertical drag resizing
function handleResizeTop(e: MouseEvent) {
  if (e.button !== 0) return;
  handleWindowInteraction();
  if ((window as any).chrome?.webview?.postMessage) {
    (window as any).chrome.webview.postMessage('wails:resize:n-resize');
  }
}

function handleResizeBottom(e: MouseEvent) {
  if (e.button !== 0) return;
  handleWindowInteraction();
  if ((window as any).chrome?.webview?.postMessage) {
    (window as any).chrome.webview.postMessage('wails:resize:s-resize');
  }
}

async function fetchLatestNotification(playSound: boolean = false) {
  const fetchId = ++activeFetchId;
  isFetchingLatest = true;
  try {
    const res = await fetch(`/api/notify/latest?t=${Date.now()}`);
    if (fetchId !== activeFetchId) return;
    if (res.ok) {
      const data = await res.json();
      if (data.className) {
        classroomName.value = data.className;
      }
      if (data.fontSize) {
        currentFontSize.value = data.fontSize;
        try {
          localStorage.setItem('desknotify_font_size', data.fontSize);
        } catch {}
      }
      const rawList: NotificationMessage[] =
        data.notifications && data.notifications.length > 0
          ? data.notifications
          : data.notification
            ? [data.notification]
            : [];

      if (data.hasNotification && rawList.length > 0) {
        notifications.value = rawList.map((item) => ({
          ...item,
          soundEnabled: data.soundEnabled,
          soundUrl: data.soundUrl,
        }));
        pruneExpiredNotifications();
        if (notifications.value.length > 0) {
          if (playSound && notifications.value[0]?.soundEnabled !== false) {
            playAlertSound(notifications.value[0]?.soundUrl);
          }
          if (notifications.value[0]?.duration > 0 && remainingSeconds.value <= 0) {
            remainingSeconds.value = notifications.value[0].duration;
            startCountdown();
          }
        }
      } else {
        notifications.value = [];
      }
    } else {
      console.warn('获取最新通知返回状态码非200:', res.status);
    }
  } catch (e) {
    console.error('获取最新通知失败', e);
  } finally {
    if (fetchId === activeFetchId) {
      isFetchingLatest = false;
    }
  }
}

const deskNotifyShowLatestHandler = async () => {
  viewMode.value = 'current';
  syncWindowHeight('current');
  await fetchLatestNotification(false);
};

const deskNotifyShowMessageHandler = (msg: NotificationMessage) => {
  handleIncomingNotification(msg);
};

// Expose handlers to Wails / window runtime
(window as any).deskNotifyShowLatest = deskNotifyShowLatestHandler;
(window as any).deskNotifyShowMessage = deskNotifyShowMessageHandler;

onMounted(() => {
  if (typeof BroadcastChannel !== 'undefined') {
    const fontChannel = new BroadcastChannel('desknotify_font_channel');
    fontChannel.onmessage = (e) => {
      if (e.data?.fontSize) {
        currentFontSize.value = e.data.fontSize;
      }
    };
  }
  window.addEventListener('storage', (e) => {
    if (e.key === 'desknotify_font_size' && e.newValue) {
      currentFontSize.value = e.newValue;
    }
  });

  connectWebSocket();

  // Process any pending commands queued before Vue mounted
  if ((window as any).__deskNotifyPendingMessage) {
    const pendingMsg = (window as any).__deskNotifyPendingMessage;
    (window as any).__deskNotifyPendingMessage = null;
    deskNotifyShowMessageHandler(pendingMsg);
  } else if ((window as any).__deskNotifyPendingShowLatest) {
    (window as any).__deskNotifyPendingShowLatest = false;
    deskNotifyShowLatestHandler();
  } else {
    fetchLatestNotification(false);
  }

  window.addEventListener('focus', () => {
    handleWindowInteraction();
    if (viewMode.value === 'current' && !isFetchingLatest) {
      fetchLatestNotification(false);
    }
  });

  window.addEventListener('pointerdown', handleWindowInteraction);
  window.addEventListener('click', handleWindowInteraction);
  window.addEventListener('resize', handleWindowResize);
  syncWindowHeight(viewMode.value);
});

onUnmounted(() => {
  window.removeEventListener('resize', handleWindowResize);
  if (resizeDebounceTimer) {
    clearTimeout(resizeDebounceTimer);
  }
});

function connectWebSocket() {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const ws = new WebSocket(`${protocol}//${window.location.host}/ws`);

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data);
      if (msg.type === 'config') {
        if (msg.fontSize) {
          currentFontSize.value = msg.fontSize;
          try {
            localStorage.setItem('desknotify_font_size', msg.fontSize);
          } catch {}
        }
        if (msg.className) {
          classroomName.value = msg.className;
        }
        return;
      }
      if (msg.fontSize) {
        currentFontSize.value = msg.fontSize;
        try {
          localStorage.setItem('desknotify_font_size', msg.fontSize);
        } catch {}
      }
      handleIncomingNotification(msg);
    } catch (e) {
      console.error('解析推送消息失败', e);
    }
  };

  ws.onclose = () => {
    setTimeout(connectWebSocket, 3000);
  };
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

function minimizeCard() {
  handleWindowInteraction();
  if ((window as any).runtime?.WindowMinimise) {
    (window as any).runtime.WindowMinimise();
  }
  fetch('/api/window/minimize', { method: 'POST' }).catch(() => {});
}

function closeCard() {
  handleWindowInteraction();
  if (timer) {
    clearInterval(timer);
    timer = null;
  }
  viewMode.value = 'current';
  lastAppliedHeight = 0;
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

async function openAttachment(att: Attachment) {
  try {
    const res = await fetch(`/api/files/${att.id}/open`, { method: 'POST' });
    if (!res.ok) {
      const err = await res.text();
      throw new Error(err || `状态码 ${res.status}`);
    }
  } catch (err: any) {
    console.error('打开文件失败:', err);
    alert('打开文件失败：' + (err.message || '系统错误'));
  }
}

async function revealAttachment(att: Attachment) {
  try {
    const res = await fetch(`/api/files/${att.id}/reveal`, { method: 'POST' });
    if (!res.ok) {
      const err = await res.text();
      throw new Error(err || `状态码 ${res.status}`);
    }
  } catch (err: any) {
    console.error('在文件夹中定位文件失败:', err);
    alert('在文件夹中定位文件失败：' + (err.message || '系统错误'));
  }
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
}

function formatExactTime(iso?: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  const hours = d.getHours().toString().padStart(2, '0');
  const minutes = d.getMinutes().toString().padStart(2, '0');
  const seconds = d.getSeconds().toString().padStart(2, '0');
  return `${hours}:${minutes}:${seconds}`;
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

.drag-header {
  -webkit-app-region: drag;
}
</style>
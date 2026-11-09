<template>
  <div class="space-y-4">
    <!-- Header with Stats & Filter & Refresh -->
    <div class="flex items-center justify-between flex-wrap gap-2 px-1">
      <div class="flex items-center gap-2">
        <h3 class="text-sm font-bold text-slate-800 flex items-center gap-1.5">
          <History class="w-4 h-4 text-blue-600" />
          <span>{{ title || '历史通知' }}</span>
        </h3>
        <span v-if="total > 0" class="text-xs px-2 py-0.5 rounded-full bg-slate-100 text-slate-600 font-medium">
          共 {{ total }} 条
        </span>
      </div>

      <div class="flex items-center gap-2">
        <!-- Filter -->
        <select
          v-model="selectedFilter"
          class="text-xs bg-white border border-slate-200 rounded-lg px-2 py-1 text-slate-600 focus:outline-none focus:ring-1 focus:ring-blue-500"
        >
          <option value="all">全部通知</option>
          <option value="pinned">置顶通知</option>
          <option value="unpinned">常规通知</option>
        </select>

        <!-- Refresh Button -->
        <button
          @click="reload"
          :disabled="loading"
          class="p-1.5 rounded-lg border border-slate-200 hover:bg-slate-100 text-slate-600 transition disabled:opacity-50"
          title="刷新列表"
        >
          <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': loading }" />
        </button>
      </div>
    </div>

    <!-- Empty State -->
    <div
      v-if="!loading && filteredItems.length === 0"
      class="bg-white rounded-2xl p-8 border border-slate-200 text-center space-y-3"
    >
      <div class="w-12 h-12 rounded-full bg-slate-100 text-slate-400 flex items-center justify-center mx-auto">
        <Inbox class="w-6 h-6" />
      </div>
      <p class="text-sm font-medium text-slate-600">暂无历史通知</p>
      <p class="text-xs text-slate-400">发布的通知与附件将持久保存在这里</p>
    </div>

    <!-- Notifications List -->
    <div v-else class="space-y-4">
      <article
        v-for="item in filteredItems"
        :key="item.id"
        class="bg-white rounded-2xl p-4 sm:p-5 border border-slate-200/90 shadow-2xs hover:shadow-xs transition space-y-3.5"
      >
        <!-- Top Metadata: Level & Time Point -->
        <div class="flex items-center justify-between text-xs border-b border-slate-100 pb-2.5">
          <div class="flex items-center gap-2">
            <span
              class="px-2.5 py-0.5 rounded-full font-semibold text-[11px] shadow-2xs flex items-center gap-1"
              :class="item.pinned ? 'bg-amber-50 text-amber-700 border border-amber-200' : 'bg-slate-50 text-slate-700 border border-slate-200'"
            >
              <Pin v-if="item.pinned" class="w-3 h-3 text-amber-600" />
              <span>{{ item.pinned ? '置顶显示' : '常规通知' }}</span>
            </span>
          </div>

          <div class="flex items-center gap-2 text-[11px]">
            <span v-if="item.duration > 0" class="text-slate-400">自动隐去: {{ item.duration }}s</span>
            <span class="text-slate-600 font-mono font-medium flex items-center gap-1 bg-slate-50 border border-slate-200/80 px-2.5 py-0.5 rounded-md">
              <Clock class="w-3 h-3 text-slate-400" />
              <span>{{ formatTime(item.createdAt) }}</span>
            </span>
          </div>
        </div>

        <!-- Full Markdown Content (No truncation) -->
        <div
          v-if="item.content"
          class="markdown-content text-slate-800 leading-relaxed"
          :style="{ fontSize: currentFontSize }"
          v-html="renderMarkdown(item.content)"
        ></div>

        <!-- Image Gallery (Grid with click-to-preview lightbox) -->
        <div v-if="item.images && item.images.length > 0" class="pt-1">
          <p class="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2 flex items-center gap-1">
            <ImageIcon class="w-3.5 h-3.5 text-blue-500" />
            <span>图片 ({{ item.images.length }}张 · 点击放大预览)</span>
          </p>
          <div
            class="grid gap-2.5"
            :class="{
              'grid-cols-1 sm:grid-cols-2': item.images.length === 1,
              'grid-cols-2 sm:grid-cols-3': item.images.length === 2,
              'grid-cols-2 sm:grid-cols-3 md:grid-cols-4': item.images.length >= 3,
            }"
          >
            <div
              v-for="(imgUrl, idx) in item.images"
              :key="idx"
              @click="openLightbox(imgUrl)"
              class="group relative rounded-xl overflow-hidden border border-slate-200 bg-slate-50 aspect-video sm:aspect-4/3 cursor-pointer hover:border-blue-400 transition shadow-2xs"
            >
              <img
                :src="imgUrl"
                class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                loading="lazy"
                alt="通知图片"
              />
              <div class="absolute inset-0 bg-black/0 group-hover:bg-black/30 transition-colors flex items-center justify-center opacity-0 group-hover:opacity-100">
                <span class="px-2.5 py-1 rounded-full bg-white/90 text-slate-800 text-xs font-semibold shadow-sm flex items-center gap-1">
                  <ZoomIn class="w-3.5 h-3.5 text-blue-600" />
                  <span>放大</span>
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- Attachments List -->
        <div v-if="item.attachments && item.attachments.length > 0" class="pt-1">
          <p class="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-2 flex items-center gap-1">
            <Paperclip class="w-3.5 h-3.5 text-blue-500" />
            <span>课件与附件 ({{ item.attachments.length }}个)</span>
          </p>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
            <div
              v-for="att in item.attachments"
              :key="att.id"
              class="flex items-center justify-between p-2.5 rounded-xl bg-slate-50 hover:bg-blue-50/50 border border-slate-200 hover:border-blue-200 text-xs transition group select-none cursor-pointer"
              @click="openAttachment(att)"
              :title="'点击直接打开：' + att.name"
            >
              <div class="flex items-center space-x-2.5 truncate min-w-0 pr-2">
                <div class="w-8 h-8 rounded-lg bg-blue-100 text-blue-600 flex items-center justify-center flex-shrink-0 group-hover:scale-105 transition-transform">
                  <FileText class="w-4 h-4" />
                </div>
                <div class="truncate">
                  <p class="font-medium text-slate-700 truncate group-hover:text-blue-600 transition">{{ att.name }}</p>
                  <p class="text-[11px] text-slate-400">{{ formatSize(att.size) }}</p>
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
      </article>
    </div>

    <!-- Infinite Scroll Sentinel & Loading Indicator -->
    <div ref="sentinelRef" class="py-4 text-center">
      <div v-if="loadingMore || (loading && items.length === 0)" class="flex items-center justify-center gap-2 text-xs text-slate-500">
        <Loader2 class="w-4 h-4 animate-spin text-blue-600" />
        <span>正在加载历史通知...</span>
      </div>
      <div v-else-if="!hasMore && items.length > 0" class="text-xs text-slate-400 py-2 flex items-center justify-center gap-1.5">
        <CheckCircle2 class="w-3.5 h-3.5 text-emerald-500" />
        <span>已加载全部历史消息（共 {{ total }} 条）</span>
      </div>
      <button
        v-else-if="hasMore && !loadingMore"
        @click="loadMore"
        class="text-xs text-blue-600 hover:text-blue-700 font-medium py-1 px-3 rounded-lg hover:bg-blue-50 transition"
      >
        点击加载更多
      </button>
    </div>

    <!-- Image Lightbox Modal -->
    <ImageLightbox :image-url="activeLightboxImage" @close="activeLightboxImage = null" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue';
import {
  History, RefreshCw, Inbox, ImageIcon, Paperclip,
  FileText, Folder, ExternalLink, ZoomIn, Loader2, CheckCircle2, Pin, Clock
} from 'lucide-vue-next';
import { marked } from 'marked';
import type { NotificationMessage, Attachment } from '../types';
import ImageLightbox from './ImageLightbox.vue';
import { spacePage } from '../utils/pangu';

const props = defineProps<{
  title?: string;
  pageSize?: number;
}>();

const limit = computed(() => props.pageSize || 10);
const items = ref<NotificationMessage[]>([]);
const total = ref(0);
const loading = ref(false);
const loadingMore = ref(false);
const hasMore = ref(true);
const selectedFilter = ref<'all' | 'pinned' | 'unpinned'>('all');
const activeLightboxImage = ref<string | null>(null);
const currentFontSize = ref(
  (typeof localStorage !== 'undefined' && localStorage.getItem('desknotify_font_size')) || '16px'
);

const sentinelRef = ref<HTMLDivElement | null>(null);
let observer: IntersectionObserver | null = null;

const filteredItems = computed(() => {
  if (selectedFilter.value === 'all') {
    return items.value;
  }
  if (selectedFilter.value === 'pinned') {
    return items.value.filter(it => it.pinned);
  }
  return items.value.filter(it => !it.pinned);
});

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

  fetchInitial();
  setupObserver();
});

onUnmounted(() => {
  if (observer) {
    observer.disconnect();
    observer = null;
  }
});

function setupObserver() {
  if (typeof IntersectionObserver === 'undefined') return;

  observer = new IntersectionObserver((entries) => {
    const entry = entries[0];
    if (entry && entry.isIntersecting) {
      if (hasMore.value && !loading.value && !loadingMore.value && items.value.length > 0) {
        loadMore();
      }
    }
  }, {
    rootMargin: '150px'
  });

  if (sentinelRef.value) {
    observer.observe(sentinelRef.value);
  }
}

function getAuthHeaders(): Record<string, string> {
  const headers: Record<string, string> = {};
  const token = localStorage.getItem('dn_token') || localStorage.getItem('dn_admin_token');
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  return headers;
}

async function fetchInitial() {
  loading.value = true;
  try {
    const res = await fetch(`/api/history?limit=${limit.value}&offset=0`, {
      headers: getAuthHeaders()
    });
    if (res.ok) {
      const data = await res.json();
      if (Array.isArray(data)) {
        items.value = data;
        total.value = data.length;
        hasMore.value = data.length >= limit.value;
      } else {
        items.value = data.items || [];
        total.value = data.total ?? (data.items ? data.items.length : 0);
        hasMore.value = !!data.hasMore;
      }
    }
  } catch (e) {
    console.error('加载历史记录失败', e);
  } finally {
    loading.value = false;
  }
}

async function loadMore() {
  if (!hasMore.value || loadingMore.value) return;
  loadingMore.value = true;
  const currentOffset = items.value.length;

  try {
    const res = await fetch(`/api/history?limit=${limit.value}&offset=${currentOffset}`, {
      headers: getAuthHeaders()
    });
    if (res.ok) {
      const data = await res.json();
      const newItems: NotificationMessage[] = Array.isArray(data) ? data : (data.items || []);
      
      // Deduplicate by ID
      const existingIds = new Set(items.value.map(it => it.id));
      const fresh = newItems.filter(it => !existingIds.has(it.id));
      items.value = [...items.value, ...fresh];

      if (Array.isArray(data)) {
        hasMore.value = newItems.length >= limit.value;
        total.value = items.value.length;
      } else {
        total.value = data.total ?? items.value.length;
        hasMore.value = !!data.hasMore;
      }
    }
  } catch (e) {
    console.error('加载更多历史失败', e);
  } finally {
    loadingMore.value = false;
  }
}

function reload() {
  items.value = [];
  hasMore.value = true;
  total.value = 0;
  return fetchInitial();
}

function prependItem(item: NotificationMessage) {
  // Prepend new item if not already exists
  if (!items.value.some(it => it.id === item.id)) {
    items.value = [item, ...items.value];
    total.value++;
  }
}

function renderMarkdown(content: string): string {
  if (!content) return '';
  const html = marked.parse(content, { breaks: true, gfm: true }) as string;
  nextTick(() => spacePage());
  return html;
}

function openLightbox(url: string) {
  activeLightboxImage.value = url;
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
  if (!bytes || isNaN(bytes)) return '0 B';
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
}

function formatTime(iso: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  const month = (d.getMonth() + 1).toString().padStart(2, '0');
  const day = d.getDate().toString().padStart(2, '0');
  const hours = d.getHours().toString().padStart(2, '0');
  const minutes = d.getMinutes().toString().padStart(2, '0');
  const seconds = d.getSeconds().toString().padStart(2, '0');
  return `${month}-${day} ${hours}:${minutes}:${seconds}`;
}

defineExpose({
  reload,
  prependItem,
  items,
  total
});
</script>

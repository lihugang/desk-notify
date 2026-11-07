<template>
  <div class="min-h-screen bg-slate-50 text-slate-800 pb-12">
    <!-- Header -->
    <header class="bg-white border-b border-slate-200 sticky top-0 z-30 shadow-sm">
      <div class="max-w-2xl mx-auto px-4 h-14 flex items-center justify-between">
        <div class="flex items-center space-x-2">
          <div class="w-8 h-8 rounded-lg bg-blue-600 flex items-center justify-center text-white font-bold shadow-sm">
            <Bell class="w-4 h-4" />
          </div>
          <div>
            <h1 class="font-bold text-base leading-tight text-slate-900">{{ classroomName || '班级电脑' }} · 通知投递</h1>
            <p class="text-xs text-slate-500">向教室大屏实时推送</p>
          </div>
        </div>
        <div class="flex items-center space-x-2 text-xs">
          <span :class="wsConnected ? 'bg-emerald-100 text-emerald-700' : 'bg-amber-100 text-amber-700'" class="px-2 py-0.5 rounded-full font-medium flex items-center gap-1">
            <span class="w-1.5 h-1.5 rounded-full" :class="wsConnected ? 'bg-emerald-500' : 'bg-amber-500'"></span>
            {{ wsConnected ? '大屏在线' : '连接检测中' }}
          </span>
        </div>
      </div>
    </header>

    <!-- Main Container -->
    <main class="max-w-2xl mx-auto px-4 mt-5 space-y-5">
      <!-- Password Modal (If public network requires login) -->
      <div v-if="authRequired && !authenticated" class="bg-white rounded-2xl p-6 shadow-sm border border-slate-200 text-center space-y-4">
        <div class="w-12 h-12 bg-blue-50 text-blue-600 rounded-full flex items-center justify-center mx-auto">
          <Lock class="w-6 h-6" />
        </div>
        <div>
          <h2 class="text-lg font-bold text-slate-900">访问安全验证</h2>
          <p class="text-xs text-slate-500 mt-1">请输入统一密码以进行通知投递</p>
        </div>
        <div class="flex max-w-xs mx-auto gap-2">
          <input
            v-model="passwordInput"
            type="password"
            placeholder="输入访问密码"
            class="flex-1 px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            @keyup.enter="handleLogin"
          />
          <button
            @click="handleLogin"
            :disabled="loggingIn || !passwordInput"
            class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium disabled:opacity-50 transition"
          >
            {{ loggingIn ? '验证中...' : '验证' }}
          </button>
        </div>
        <p v-if="authError" class="text-xs text-red-500">{{ authError }}</p>
      </div>

      <!-- Editor Card -->
      <div v-else class="bg-white rounded-2xl shadow-sm border border-slate-200 overflow-hidden">
        <!-- Visual Formatting Toolbar (Without exposing raw markdown) -->
        <div class="bg-slate-50 border-b border-slate-200 px-4 py-2.5 flex items-center justify-between flex-wrap gap-2 text-slate-600">
          <div class="flex items-center space-x-1">
            <button @click="applyFormat('bold')" title="加粗" class="p-1.5 hover:bg-slate-200 rounded text-slate-700 transition">
              <Bold class="w-4 h-4" />
            </button>
            <button @click="applyFormat('italic')" title="斜体" class="p-1.5 hover:bg-slate-200 rounded text-slate-700 transition">
              <Italic class="w-4 h-4" />
            </button>
            <button @click="applyFormat('h2')" title="大字号" class="p-1.5 hover:bg-slate-200 rounded text-slate-700 transition">
              <Heading2 class="w-4 h-4" />
            </button>
            <button @click="applyFormat('list')" title="列表" class="p-1.5 hover:bg-slate-200 rounded text-slate-700 transition">
              <List class="w-4 h-4" />
            </button>
            <button @click="applyFormat('quote')" title="引用高亮" class="p-1.5 hover:bg-slate-200 rounded text-slate-700 transition">
              <Quote class="w-4 h-4" />
            </button>
            <div class="h-4 w-px bg-slate-300 mx-1"></div>
            <label class="p-1.5 hover:bg-slate-200 rounded text-slate-700 cursor-pointer transition flex items-center gap-1" title="插入图片">
              <ImageIcon class="w-4 h-4" />
              <input type="file" accept="image/*" multiple class="hidden" @change="handleImageSelect" />
            </label>
            <label class="p-1.5 hover:bg-slate-200 rounded text-slate-700 cursor-pointer transition flex items-center gap-1" title="上传附件">
              <Paperclip class="w-4 h-4" />
              <input type="file" multiple class="hidden" @change="handleFileSelect" />
            </label>
          </div>

          <div class="flex items-center space-x-2">
            <!-- Toggle Preview Button -->
            <button
              @click="isPreview = !isPreview"
              type="button"
              class="px-2.5 py-1 text-xs rounded-lg border flex items-center gap-1.5 transition font-medium cursor-pointer"
              :class="isPreview ? 'bg-blue-50 border-blue-300 text-blue-600 shadow-2xs' : 'bg-slate-50 border-slate-200 text-slate-600 hover:bg-slate-100'"
              :title="isPreview ? '切换回文本编辑' : '大屏展示效果预览'"
            >
              <component :is="isPreview ? Edit3 : Eye" class="w-3.5 h-3.5" />
              <span>{{ isPreview ? '返回编辑' : '效果预览' }}</span>
            </button>

            <!-- Pinned display toggle -->
            <label
              class="flex items-center space-x-1.5 cursor-pointer text-xs select-none py-1 px-2.5 rounded-lg border transition"
              :class="isPinned ? 'bg-amber-50 border-amber-300 text-amber-800' : 'bg-slate-50 border-slate-200 text-slate-600 hover:bg-slate-100'"
              title="置顶显示：弹窗置顶并抢走焦点；不置顶：后台静默打开并闪烁任务栏"
            >
              <input
                type="checkbox"
                v-model="isPinned"
                class="w-4 h-4 rounded text-amber-600 focus:ring-amber-500 cursor-pointer"
              />
              <Pin class="w-3.5 h-3.5" :class="isPinned ? 'text-amber-600 fill-amber-500/20' : 'text-slate-400'" />
              <span class="font-medium">置顶显示</span>
            </label>
          </div>
        </div>

        <!-- Mode: Preview Mode -->
        <div v-if="isPreview" class="p-5 min-h-[260px] bg-slate-50/50 space-y-4">
          <div class="flex items-center justify-between pb-2 border-b border-slate-200/80 text-xs">
            <span class="font-medium text-slate-500 flex items-center gap-1.5">
              <Eye class="w-3.5 h-3.5 text-blue-500" />
              <span>大屏呈现效果预览</span>
            </span>
            <span
              class="px-2.5 py-0.5 rounded-full text-[11px] font-semibold"
              :class="isPinned ? 'bg-amber-500 text-white' : 'bg-blue-500 text-white'"
            >
              {{ isPinned ? '强提醒·置顶卡片' : '常规通知' }}
            </span>
          </div>

          <!-- Rendered Markdown Content -->
          <div
            class="markdown-content text-slate-800 text-base leading-relaxed bg-white p-4 rounded-xl border border-slate-200/60 shadow-2xs"
            v-html="renderedMarkdown"
          ></div>

          <!-- Preview Images Grid -->
          <div v-if="images && images.length > 0" class="pt-2">
            <p class="text-xs font-semibold text-slate-500 mb-2 flex items-center gap-1">
              <ImageIcon class="w-3.5 h-3.5 text-blue-500" />
              <span>图片 ({{ images.length }}张 · 点击放大预览)</span>
            </p>
            <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
              <div
                v-for="(img, idx) in images"
                :key="idx"
                @click="previewImage(img.preview)"
                class="group relative rounded-xl overflow-hidden border border-slate-200 bg-white aspect-video cursor-pointer hover:border-blue-400 hover:shadow-md transition shadow-2xs"
                title="点击放大预览"
              >
                <img :src="img.preview" class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300" />
                <div class="absolute inset-0 bg-black/35 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-1 text-white text-xs font-medium">
                  <ZoomIn class="w-4 h-4" />
                  <span>放大预览</span>
                </div>
              </div>
            </div>
          </div>

          <!-- Preview Attachments List -->
          <div v-if="attachments && attachments.length > 0" class="pt-2">
            <p class="text-xs font-semibold text-slate-500 mb-2 flex items-center gap-1">
              <Paperclip class="w-3.5 h-3.5 text-blue-500" />
              <span>附件 ({{ attachments.length }}个)</span>
            </p>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
              <div
                v-for="(file, idx) in attachments"
                :key="idx"
                class="flex items-center justify-between p-2.5 bg-white border border-slate-200 rounded-xl text-xs"
              >
                <div class="flex items-center space-x-2 truncate">
                  <FileText class="w-4 h-4 text-blue-600 flex-shrink-0" />
                  <span class="truncate font-medium text-slate-700">{{ file.name }}</span>
                  <span class="text-slate-400">({{ formatSize(file.size) }})</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Mode: Textarea / Paste Area (Edit Mode) -->
        <div v-else class="p-4 sm:p-5" @paste="handlePaste">
          <textarea
            ref="textareaRef"
            v-model="textContent"
            rows="10"
            placeholder="请输入要投递给班级大屏的通知正文...（可直接按 Ctrl+V 粘贴剪贴板截图）"
            class="w-full text-base placeholder-slate-400 focus:outline-none resize-y min-h-[240px] leading-relaxed text-slate-800"
          ></textarea>

          <!-- Uploaded Images Grid -->
          <div v-if="images && images.length > 0" class="mt-4 pt-3 border-t border-slate-100">
            <div class="flex items-center justify-between mb-2">
              <span class="text-xs font-semibold text-slate-500 flex items-center gap-1">
                <ImageIcon class="w-3.5 h-3.5 text-blue-600" />
                <span>已添加图片 ({{ images.length }}张 · 点击图片可放大预览)</span>
              </span>
            </div>
            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3">
              <div
                v-for="(img, idx) in images"
                :key="idx"
                @click="previewImage(img.preview)"
                class="group relative rounded-xl overflow-hidden border border-slate-200 bg-slate-50 aspect-video cursor-pointer hover:border-blue-400 hover:shadow-md transition shadow-2xs"
                title="点击放大预览"
              >
                <img :src="img.preview" class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300" />
                <!-- Hover overlay -->
                <div class="absolute inset-0 bg-black/35 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-1 text-white text-xs font-medium">
                  <ZoomIn class="w-4 h-4" />
                  <span>预览</span>
                </div>
                <!-- Remove button -->
                <button
                  type="button"
                  @click.stop="removeImage(idx)"
                  class="absolute top-1.5 right-1.5 bg-black/60 hover:bg-red-600 text-white rounded-full p-1 opacity-90 transition z-10"
                  title="移除此图片"
                >
                  <X class="w-3.5 h-3.5" />
                </button>
                <!-- Image name / size bar -->
                <div class="absolute bottom-0 inset-x-0 bg-gradient-to-t from-black/70 to-transparent p-1.5 text-[10px] text-white/90 truncate pointer-events-none opacity-0 group-hover:opacity-100 transition-opacity">
                  {{ img.name }} ({{ formatSize(img.size) }})
                </div>
              </div>
            </div>
          </div>

          <!-- Uploaded Attachments List -->
          <div v-if="attachments && attachments.length > 0" class="mt-4 pt-3 border-t border-slate-100 space-y-2">
            <div class="text-xs font-semibold text-slate-500 mb-1 flex items-center gap-1">
              <Paperclip class="w-3.5 h-3.5 text-blue-600" />
              <span>已添加附件 ({{ attachments.length }}个)</span>
            </div>
            <div
              v-for="(file, idx) in attachments"
              :key="idx"
              class="flex items-center justify-between p-2.5 bg-slate-50 border border-slate-200 rounded-xl text-xs"
            >
              <div class="flex items-center space-x-2 truncate">
                <FileText class="w-4 h-4 text-blue-600 flex-shrink-0" />
                <span class="truncate font-medium text-slate-700">{{ file.name }}</span>
                <span class="text-slate-400">({{ formatSize(file.size) }})</span>
              </div>
              <button @click="removeAttachment(idx)" class="text-slate-400 hover:text-red-500 p-1" title="删除附件">
                <X class="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        </div>

        <!-- Editor Footer & Submit -->
        <div class="bg-slate-50/80 border-t border-slate-200 px-4 py-3 flex items-center justify-between">
          <div class="text-xs text-slate-500 flex items-center gap-1.5">
            <span class="inline-block w-2 h-2 rounded-full" :class="isPinned ? 'bg-amber-500' : 'bg-blue-500'"></span>
            <span>{{ isPinned ? '桌面置顶弹窗（强提醒·抢占焦点）' : '常规通知（后台默默打开·任务栏闪烁）' }}</span>
          </div>

          <button
            @click="handleSubmit"
            :disabled="submitting || (!textContent.trim() && (!images || images.length === 0) && (!attachments || attachments.length === 0))"
            class="px-5 py-2 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-xl shadow-sm text-sm flex items-center gap-1.5 disabled:opacity-50 transition"
          >
            <Send class="w-4 h-4" />
            <span>{{ submitting ? '正在投递...' : '投递到大屏' }}</span>
          </button>
        </div>
      </div>

      <!-- Delivery History with Infinite Scroll & Full Details -->
      <section v-if="authenticated" class="mt-8 pt-4 border-t border-slate-200">
        <HistoryList ref="historyListRef" title="投递历史记录" :page-size="10" />
      </section>
    </main>

    <!-- Image Lightbox Modal -->
    <ImageLightbox :image-url="activeLightboxImage" @close="activeLightboxImage = null" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue';
import {
  Bell, Lock, Bold, Italic, Heading2, List, Quote,
  ImageIcon, Paperclip, Send, X, FileText, Pin, ZoomIn, Eye, Edit3
} from 'lucide-vue-next';
import { marked } from 'marked';
import HistoryList from '../components/HistoryList.vue';
import ImageLightbox from '../components/ImageLightbox.vue';

interface UploadItem {
  file: File;
  preview?: string;
  name: string;
  size: number;
}

const classroomName = ref('班级电脑');
const wsConnected = ref(false);
const authRequired = ref(false);
const authenticated = ref(true);
const passwordInput = ref('');
const authError = ref('');
const loggingIn = ref(false);

const textContent = ref('');
const isPinned = ref(true);
const duration = ref(0);
const images = ref<UploadItem[]>([]);
const attachments = ref<UploadItem[]>([]);
const submitting = ref(false);
const historyListRef = ref<InstanceType<typeof HistoryList> | null>(null);
const textareaRef = ref<HTMLTextAreaElement | null>(null);

const activeLightboxImage = ref<string | null>(null);
const isPreview = ref(false);

const renderedMarkdown = computed(() => {
  if (!textContent.value.trim()) {
    return '<p class="text-slate-400 italic">（暂无正文内容）</p>';
  }
  return marked.parse(textContent.value, { breaks: true, gfm: true }) as string;
});

function previewImage(url?: string) {
  if (url) {
    activeLightboxImage.value = url;
  }
}

onMounted(() => {
  checkAuthAndStatus();
});

onUnmounted(() => {
  images.value.forEach(img => {
    if (img.preview) URL.revokeObjectURL(img.preview);
  });
});

async function checkAuthAndStatus() {
  try {
    const res = await fetch('/api/status');
    if (res.status === 401) {
      authRequired.value = true;
      authenticated.value = false;
      return;
    }
    const data = await res.json();
    classroomName.value = data.className || '2907';
    wsConnected.value = true;
    authenticated.value = true;
  } catch (e) {
    wsConnected.value = false;
  }
}

async function handleLogin() {
  if (!passwordInput.value) return;
  loggingIn.value = true;
  authError.value = '';
  try {
    const res = await fetch('/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: passwordInput.value })
    });
    if (res.ok) {
      authenticated.value = true;
      authRequired.value = false;
      historyListRef.value?.reload();
    } else {
      authError.value = '密码错误，请重新输入';
    }
  } catch (e) {
    authError.value = '网络错误，请稍后再试';
  } finally {
    loggingIn.value = false;
  }
}

function applyFormat(type: 'bold' | 'italic' | 'h2' | 'list' | 'quote') {
  const el = textareaRef.value;
  if (!el) return;
  const start = el.selectionStart;
  const end = el.selectionEnd;
  const sel = textContent.value.substring(start, end) || '文本';
  let replacement = '';

  switch (type) {
    case 'bold': replacement = `**${sel}**`; break;
    case 'italic': replacement = `*${sel}*`; break;
    case 'h2': replacement = `\n## ${sel}\n`; break;
    case 'list': replacement = `\n- ${sel}\n`; break;
    case 'quote': replacement = `\n> ${sel}\n`; break;
  }

  textContent.value = textContent.value.substring(0, start) + replacement + textContent.value.substring(end);
}

function handlePaste(e: ClipboardEvent) {
  const items = e.clipboardData?.items;
  if (!items) return;
  for (let i = 0; i < items.length; i++) {
    if (items[i].type.indexOf('image') !== -1) {
      const file = items[i].getAsFile();
      if (file) {
        images.value.push({
          file,
          preview: URL.createObjectURL(file),
          name: file.name || `screenshot_${Date.now()}.png`,
          size: file.size
        });
      }
    }
  }
}

function handleImageSelect(e: Event) {
  const target = e.target as HTMLInputElement;
  if (!target.files) return;
  Array.from(target.files).forEach(file => {
    images.value.push({
      file,
      preview: URL.createObjectURL(file),
      name: file.name,
      size: file.size
    });
  });
  target.value = '';
}

function handleFileSelect(e: Event) {
  const target = e.target as HTMLInputElement;
  if (!target.files) return;
  Array.from(target.files).forEach(file => {
    attachments.value.push({
      file,
      name: file.name,
      size: file.size
    });
  });
  target.value = '';
}

function removeImage(idx: number) {
  const item = images.value[idx];
  if (item?.preview) {
    if (activeLightboxImage.value === item.preview) {
      activeLightboxImage.value = null;
    }
    URL.revokeObjectURL(item.preview);
  }
  images.value.splice(idx, 1);
}

function removeAttachment(idx: number) {
  attachments.value.splice(idx, 1);
}

async function handleSubmit() {
  if (!textContent.value.trim() && (!images.value || images.value.length === 0) && (!attachments.value || attachments.value.length === 0)) return;
  submitting.value = true;

  try {
    const formData = new FormData();
    formData.append('content', textContent.value);
    formData.append('pinned', isPinned.value ? 'true' : 'false');
    formData.append('duration', duration.value.toString());

    images.value.forEach(img => {
      formData.append('images', img.file);
    });
    attachments.value.forEach(att => {
      formData.append('attachments', att.file);
    });

    const res = await fetch('/api/notify', {
      method: 'POST',
      body: formData
    });

    if (res.ok) {
      images.value.forEach(img => {
        if (img.preview) URL.revokeObjectURL(img.preview);
      });
      textContent.value = '';
      images.value = [];
      attachments.value = [];
      isPreview.value = false;
      historyListRef.value?.reload();
      alert('已成功投递到班级大屏！');
    } else {
      alert('投递失败，请重试');
    }
  } catch (e) {
    alert('投递失败：网络异常');
  } finally {
    submitting.value = false;
  }
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
}
</script>

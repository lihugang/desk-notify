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
          <h2 class="text-lg font-bold text-slate-900">公网访问安全验证</h2>
          <p class="text-xs text-slate-500 mt-1">当前通过公网/外部通道访问，请输入教师投递口令</p>
        </div>
        <div class="flex max-w-xs mx-auto gap-2">
          <input
            v-model="passwordInput"
            type="password"
            placeholder="输入访问口令"
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
            <!-- Urgent level selection -->
            <select v-model="messageLevel" class="text-xs bg-white border border-slate-200 rounded-md px-2 py-1 focus:ring-1 focus:ring-blue-500">
              <option value="info">普通通知</option>
              <option value="important">重要通知</option>
              <option value="urgent">紧急提醒</option>
            </select>
          </div>
        </div>

        <!-- Textarea / Paste Area -->
        <div class="p-4" @paste="handlePaste">
          <textarea
            ref="textareaRef"
            v-model="textContent"
            rows="5"
            placeholder="请输入要投递给班级大屏的通知正文...（可直接按 Ctrl+V 粘贴剪贴板截图）"
            class="w-full text-base placeholder-slate-400 focus:outline-none resize-none leading-relaxed"
          ></textarea>

          <!-- Uploaded Images Grid -->
          <div v-if="images && images.length > 0" class="mt-3 grid grid-cols-3 sm:grid-cols-4 gap-2">
            <div v-for="(img, idx) in images" :key="idx" class="relative group rounded-lg overflow-hidden border border-slate-200 aspect-video bg-slate-100">
              <img :src="img.preview" class="w-full h-full object-cover" />
              <button
                @click="removeImage(idx)"
                class="absolute top-1 right-1 bg-black/60 hover:bg-black/80 text-white rounded-full p-1 opacity-90 transition"
              >
                <X class="w-3 h-3" />
              </button>
            </div>
          </div>

          <!-- Uploaded Attachments List -->
          <div v-if="attachments && attachments.length > 0" class="mt-3 space-y-2">
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
              <button @click="removeAttachment(idx)" class="text-slate-400 hover:text-red-500 p-1">
                <X class="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        </div>

        <!-- Editor Footer & Submit -->
        <div class="bg-slate-50/80 border-t border-slate-200 px-4 py-3 flex items-center justify-between">
          <div class="text-xs text-slate-400 flex items-center gap-1.5">
            <span class="inline-block w-2 h-2 rounded-full bg-emerald-500"></span>
            <span>桌面置顶卡片（需手动点击关闭）</span>
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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import {
  Bell, Lock, Bold, Italic, Heading2, List, Quote,
  ImageIcon, Paperclip, Send, X, FileText
} from 'lucide-vue-next';
import HistoryList from '../components/HistoryList.vue';

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
const messageLevel = ref<'info' | 'important' | 'urgent'>('info');
const duration = ref(0);
const images = ref<UploadItem[]>([]);
const attachments = ref<UploadItem[]>([]);
const submitting = ref(false);
const historyListRef = ref<InstanceType<typeof HistoryList> | null>(null);
const textareaRef = ref<HTMLTextAreaElement | null>(null);

onMounted(() => {
  checkAuthAndStatus();
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
      authError.value = '口令错误，请重新输入';
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
    formData.append('level', messageLevel.value);
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
      textContent.value = '';
      images.value = [];
      attachments.value = [];
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

<template>
  <div class="min-h-screen bg-slate-50 text-slate-800 p-6">
    <div class="max-w-xl mx-auto bg-white rounded-2xl shadow-sm border border-slate-200 overflow-hidden">
      <!-- Tabs -->
      <div class="flex border-b border-slate-200 bg-slate-50/50">
        <button
          @click="activeTab = 'qrcode'"
          :class="activeTab === 'qrcode' ? 'border-b-2 border-blue-600 text-blue-600 bg-white' : 'text-slate-500 hover:text-slate-700'"
          class="flex-1 py-3 text-sm font-semibold text-center transition flex items-center justify-center gap-1.5"
        >
          <QrCode class="w-4 h-4" />
          <span>扫码投递 (手机端)</span>
        </button>
        <button
          @click="activeTab = 'config'"
          :class="activeTab === 'config' ? 'border-b-2 border-blue-600 text-blue-600 bg-white' : 'text-slate-500 hover:text-slate-700'"
          class="flex-1 py-3 text-sm font-semibold text-center transition flex items-center justify-center gap-1.5"
        >
          <Settings class="w-4 h-4" />
          <span>参数与穿透设置</span>
        </button>
      </div>

      <!-- Tab 1: QR Code & LAN Links -->
      <div v-if="activeTab === 'qrcode'" class="p-6 text-center space-y-5">
        <div>
          <h2 class="text-base font-bold text-slate-900">{{ config.className || '三年二班' }} · 投递入口</h2>
          <p class="text-xs text-slate-500 mt-0.5">请老师使用手机微信或浏览器扫描下方二维码</p>
        </div>

        <div class="flex justify-center">
          <div class="p-3 bg-white border-2 border-slate-100 rounded-2xl shadow-sm inline-block">
            <canvas ref="qrCanvasRef" class="w-48 h-48 rounded-lg"></canvas>
          </div>
        </div>

        <!-- Connection Addresses -->
        <div class="space-y-2 text-left bg-slate-50 p-4 rounded-xl border border-slate-200">
          <p class="text-xs font-bold text-slate-400 uppercase tracking-wider">可选连接地址</p>
          <div v-for="(ip, idx) in lanIPs" :key="'http-'+idx" class="flex items-center justify-between text-xs py-1">
            <span class="text-slate-600 font-mono">http://{{ ip }}:{{ config.port }}</span>
            <button @click="copyUrl(`http://${ip}:${config.port}`)" class="text-blue-600 hover:underline">复制链接</button>
          </div>
          <div v-for="(ip, idx) in lanIPs" :key="'https-'+idx" class="flex items-center justify-between text-xs py-1 border-t border-slate-200/50 pt-1">
            <span class="text-indigo-600 font-mono flex items-center gap-1">
              <span class="px-1 py-0.5 bg-indigo-50 text-[10px] rounded font-medium">HTTPS</span>
              https://{{ ip }}:{{ (config.port || 8920) + 1 }}
            </span>
            <button @click="copyUrl(`https://${ip}:${(config.port || 8920) + 1}`)" class="text-indigo-600 hover:underline">复制</button>
          </div>
          <div v-if="frpPublicUrl" class="flex items-center justify-between text-xs py-1 border-t border-slate-200/60 pt-1.5">
            <span class="text-emerald-600 font-medium font-mono truncate mr-2">公网: {{ frpPublicUrl }}</span>
            <button @click="copyUrl(frpPublicUrl)" class="text-blue-600 hover:underline flex-shrink-0">复制链接</button>
          </div>

          <!-- Root CA Certificate Info & Download -->
          <div class="mt-3 pt-3 border-t border-slate-200/80 flex items-center justify-between text-xs">
            <div class="text-[11px] text-slate-500">
              <span>手机/电脑导入根证书，可消除警告并安装 PWA</span>
            </div>
            <a
              href="/DeskNotify-CA.cer"
              download="DeskNotify-CA.cer"
              class="px-2.5 py-1 bg-white hover:bg-slate-100 border border-slate-200 rounded text-slate-700 text-xs font-medium shadow-2xs transition inline-flex items-center gap-1"
            >
              <span>下载根证书 (.cer)</span>
            </a>
          </div>
        </div>
      </div>

      <!-- Tab 2: Settings Form -->
      <div v-if="activeTab === 'config'" class="p-6 space-y-4">
        <div>
          <label class="block text-xs font-semibold text-slate-700 mb-1">班级名称</label>
          <input
            v-model="config.className"
            type="text"
            class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="如：高一(3)班"
          />
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="block text-xs font-semibold text-slate-700 mb-1">本地服务端口</label>
            <input
              v-model.number="config.port"
              type="number"
              class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label class="block text-xs font-semibold text-slate-700 mb-1">公网投递口令 (密码)</label>
            <input
              v-model="config.publicPassword"
              type="text"
              class="w-full px-3 py-2 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="外网访问强制验证"
            />
          </div>
        </div>

        <div class="pt-2 border-t border-slate-100 space-y-3">
          <!-- Notification Sound Settings Section -->
          <div class="flex items-center justify-between">
            <div>
              <span class="text-xs font-semibold text-slate-700">桌面弹窗提示音</span>
              <p class="text-[11px] text-slate-400 mt-0.5">收到新通知时，教室大屏自动播放音效提醒</p>
            </div>
            <input v-model="config.soundEnabled" type="checkbox" class="w-4 h-4 rounded text-blue-600 cursor-pointer" />
          </div>

          <div v-if="config.soundEnabled" class="space-y-3 bg-slate-50 p-3.5 rounded-xl border border-slate-200">
            <div class="flex items-center justify-between">
              <label class="block text-xs font-semibold text-slate-700">音效来源与风格</label>
              <button
                type="button"
                @click="previewSound"
                class="inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-lg transition shadow-xs"
                :class="isPlayingPreview ? 'bg-amber-100 text-amber-800 hover:bg-amber-200 border border-amber-200' : 'bg-blue-50 text-blue-700 hover:bg-blue-100 border border-blue-200'"
              >
                <Volume2 v-if="!isPlayingPreview" class="w-3.5 h-3.5" />
                <VolumeX v-else class="w-3.5 h-3.5 animate-pulse" />
                <span>{{ isPlayingPreview ? '停止试听' : '试听当前音效' }}</span>
              </button>
            </div>

            <!-- Mode Selector: Default vs Preset vs Custom -->
            <div class="grid grid-cols-3 gap-2">
              <button
                type="button"
                @click="setSoundMode('default')"
                :class="soundMode === 'default' ? 'bg-white border-blue-600 text-blue-600 font-semibold shadow-xs ring-1 ring-blue-600' : 'bg-white/70 border-slate-200 text-slate-600 hover:bg-white'"
                class="py-2 px-2 text-xs rounded-lg border text-center transition"
              >
                默认内嵌音效
              </button>
              <button
                type="button"
                @click="setSoundMode('preset')"
                :class="soundMode === 'preset' ? 'bg-white border-blue-600 text-blue-600 font-semibold shadow-xs ring-1 ring-blue-600' : 'bg-white/70 border-slate-200 text-slate-600 hover:bg-white'"
                class="py-2 px-2 text-xs rounded-lg border text-center transition"
              >
                内嵌风格预设
              </button>
              <button
                type="button"
                @click="setSoundMode('custom')"
                :class="soundMode === 'custom' ? 'bg-white border-blue-600 text-blue-600 font-semibold shadow-xs ring-1 ring-blue-600' : 'bg-white/70 border-slate-200 text-slate-600 hover:bg-white'"
                class="py-2 px-2 text-xs rounded-lg border text-center transition"
              >
                自定义音频
              </button>
            </div>

            <!-- Mode 1: Default -->
            <div v-if="soundMode === 'default'" class="text-xs text-slate-600 bg-white p-3 rounded-lg border border-slate-200/80 space-y-1">
              <div class="flex items-center justify-between">
                <span class="font-semibold text-slate-800">系统默认提示音 (清脆风铃)</span>
                <span class="px-2 py-0.5 bg-blue-50 text-blue-600 rounded text-[10px] font-medium">已内嵌</span>
              </div>
              <p class="text-[11px] text-slate-400">已完整内嵌于单一可执行程序中，绿色免安装，开箱即播。</p>
            </div>

            <!-- Mode 2: Presets -->
            <div v-if="soundMode === 'preset'" class="space-y-2 bg-white p-3 rounded-lg border border-slate-200/80">
              <label class="block text-[11px] font-medium text-slate-500">选择内置精选音效</label>
              <div class="grid grid-cols-2 gap-2">
                <button
                  v-for="preset in presets"
                  :key="preset.id"
                  type="button"
                  @click="selectPreset(preset.id)"
                  :class="config.customSound === preset.id ? 'bg-blue-50 border-blue-500 text-blue-700 font-semibold ring-1 ring-blue-400 shadow-xs' : 'bg-slate-50 border-slate-200 text-slate-600 hover:bg-slate-100'"
                  class="flex items-center justify-between px-2.5 py-2 rounded-lg border text-xs text-left transition"
                >
                  <span>{{ preset.label }}</span>
                  <span class="text-[10px] text-slate-400 font-normal">{{ preset.desc }}</span>
                </button>
              </div>
            </div>

            <!-- Mode 3: Custom Upload & Input -->
            <div v-if="soundMode === 'custom'" class="space-y-3 bg-white p-3 rounded-lg border border-slate-200/80">
              <div>
                <label class="block text-[11px] font-medium text-slate-600 mb-1.5">上传自定义音频 (支持 MP3、WAV、OGG)</label>
                <div class="flex items-center gap-2">
                  <input
                    ref="audioFileInputRef"
                    type="file"
                    accept="audio/mp3,audio/wav,audio/ogg,audio/mpeg,.mp3,.wav,.ogg,.m4a"
                    class="hidden"
                    @change="handleFileUpload"
                  />
                  <button
                    type="button"
                    @click="audioFileInputRef?.click()"
                    :disabled="uploadingSound"
                    class="px-3 py-1.5 bg-slate-100 hover:bg-slate-200 text-slate-700 rounded-lg text-xs font-medium border border-slate-200 transition flex items-center gap-1.5 shadow-xs"
                  >
                    <Upload class="w-3.5 h-3.5" />
                    <span>{{ uploadingSound ? '正在上传...' : '选择本地音频上传' }}</span>
                  </button>
                  <span v-if="uploadedFileName" class="text-xs text-emerald-600 font-medium truncate flex-1">
                    当前: {{ uploadedFileName }}
                  </span>
                </div>
              </div>

              <div>
                <label class="block text-[11px] font-medium text-slate-500 mb-1">或直接指定音频文件路径 / URL</label>
                <div class="flex gap-2">
                  <input
                    v-model="config.customSound"
                    type="text"
                    placeholder="如 C:\Sounds\bell.mp3 或 http://..."
                    class="flex-1 px-2.5 py-1.5 bg-slate-50 border border-slate-200 rounded text-xs focus:ring-1 focus:ring-blue-500 focus:bg-white"
                  />
                  <button
                    v-if="config.customSound"
                    type="button"
                    @click="resetToDefaultSound"
                    class="px-2.5 py-1 text-xs text-slate-500 hover:text-red-600 hover:bg-red-50 rounded border border-slate-200 transition flex items-center gap-1"
                    title="恢复默认提示音"
                  >
                    <RotateCcw class="w-3 h-3" />
                    <span>恢复默认</span>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="pt-2 border-t border-slate-100 space-y-3">
          <div class="flex items-center justify-between">
            <span class="text-xs font-semibold text-slate-700">启用 frp 公网穿透</span>
            <input v-model="config.frp.enabled" type="checkbox" class="w-4 h-4 rounded text-blue-600" />
          </div>

          <div v-if="config.frp.enabled" class="space-y-3 bg-slate-50 p-3.5 rounded-xl border border-slate-200">
            <div class="grid grid-cols-3 gap-2">
              <div class="col-span-2">
                <label class="block text-[11px] font-medium text-slate-500 mb-1">frps 服务器地址</label>
                <input
                  v-model="config.frp.serverAddr"
                  type="text"
                  placeholder="frp.example.com"
                  class="w-full px-2.5 py-1.5 bg-white border border-slate-200 rounded text-xs focus:ring-1 focus:ring-blue-500"
                />
              </div>
              <div>
                <label class="block text-[11px] font-medium text-slate-500 mb-1">服务器端口</label>
                <input
                  v-model.number="config.frp.serverPort"
                  type="number"
                  placeholder="7000"
                  class="w-full px-2.5 py-1.5 bg-white border border-slate-200 rounded text-xs focus:ring-1 focus:ring-blue-500"
                />
              </div>
            </div>

            <div class="grid grid-cols-2 gap-2">
              <div>
                <label class="block text-[11px] font-medium text-slate-500 mb-1">Auth Token</label>
                <input
                  v-model="config.frp.token"
                  type="password"
                  placeholder="通信密钥"
                  class="w-full px-2.5 py-1.5 bg-white border border-slate-200 rounded text-xs focus:ring-1 focus:ring-blue-500"
                />
              </div>
              <div>
                <label class="block text-[11px] font-medium text-slate-500 mb-1">远程绑定端口</label>
                <input
                  v-model.number="config.frp.remotePort"
                  type="number"
                  placeholder="如 8921"
                  class="w-full px-2.5 py-1.5 bg-white border border-slate-200 rounded text-xs focus:ring-1 focus:ring-blue-500"
                />
              </div>
            </div>
          </div>
        </div>

        <div class="pt-3 flex justify-end space-x-2">
          <button
            @click="saveConfig"
            :disabled="saving"
            class="px-5 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-xl text-sm font-semibold shadow-xs disabled:opacity-50 transition"
          >
            {{ saving ? '保存中...' : '保存配置并生效' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue';
import { QrCode, Settings, Volume2, VolumeX, Upload, RotateCcw } from 'lucide-vue-next';
import QRCode from 'qrcode';
import type { AppConfig } from '../types';

const activeTab = ref<'qrcode' | 'config'>('qrcode');
const qrCanvasRef = ref<HTMLCanvasElement | null>(null);
const lanIPs = ref<string[]>(['127.0.0.1']);
const frpPublicUrl = ref('');
const saving = ref(false);

const audioFileInputRef = ref<HTMLInputElement | null>(null);
const uploadingSound = ref(false);
const uploadedFileName = ref('');
const soundMode = ref<'default' | 'preset' | 'custom'>('default');
const isPlayingPreview = ref(false);
let previewAudio: HTMLAudioElement | null = null;

const presets = [
  { id: 'glass.mp3', label: '清脆风铃', desc: '灵动通透' },
  { id: 'minimal.mp3', label: '极简提示', desc: '干脆利落' },
  { id: 'soft.mp3', label: '柔和气泡', desc: '温润悦耳' },
  { id: 'polite.mp3', label: '优雅轻响', desc: '经典现代' },
  { id: 'chime.mp3', label: '空灵旋律', desc: '舒缓悠扬' },
];

const config = ref<AppConfig>({
  className: '高一(3)班',
  port: 8920,
  publicPassword: '',
  lanAuthRequired: false,
  defaultDuration: 30,
  soundEnabled: true,
  customSound: '',
  autoStart: true,
  frp: {
    enabled: false,
    serverAddr: '',
    serverPort: 7000,
    token: '',
    remotePort: 8920
  }
});

onMounted(async () => {
  await fetchStatusAndConfig();
  renderQRCode();
});

async function fetchStatusAndConfig() {
  try {
    const res = await fetch('/api/config');
    if (res.ok) {
      const data = await res.json();
      config.value = data.config;
      lanIPs.value = data.lanIPs || ['127.0.0.1'];
      frpPublicUrl.value = data.frpPublicUrl || '';

      // Initialize soundMode based on loaded config
      initSoundMode();
    }
  } catch (e) {
    console.warn('获取配置失败', e);
  }
}

function initSoundMode() {
  const current = (config.value.customSound || '').trim();
  if (!current || current === 'default') {
    soundMode.value = 'default';
  } else if (presets.some(p => p.id === current || p.id === current.replace(/^sounds\//, ''))) {
    soundMode.value = 'preset';
    config.value.customSound = current.replace(/^sounds\//, '');
  } else {
    soundMode.value = 'custom';
    uploadedFileName.value = current;
  }
}

function setSoundMode(mode: 'default' | 'preset' | 'custom') {
  soundMode.value = mode;
  stopPreview();

  if (mode === 'default') {
    config.value.customSound = '';
  } else if (mode === 'preset') {
    if (!presets.some(p => p.id === config.value.customSound)) {
      config.value.customSound = 'glass.mp3';
    }
  } else if (mode === 'custom') {
    if (!config.value.customSound || presets.some(p => p.id === config.value.customSound)) {
      config.value.customSound = uploadedFileName.value || '';
    }
  }
}

function selectPreset(presetId: string) {
  config.value.customSound = presetId;
  stopPreview();
}

function resetToDefaultSound() {
  config.value.customSound = '';
  soundMode.value = 'default';
  uploadedFileName.value = '';
  stopPreview();
}

async function handleFileUpload(e: Event) {
  const target = e.target as HTMLInputElement;
  if (!target.files || target.files.length === 0) return;

  const file = target.files[0];
  const formData = new FormData();
  formData.append('audio', file);

  uploadingSound.value = true;
  try {
    const res = await fetch('/api/sound/upload', {
      method: 'POST',
      body: formData,
    });
    if (res.ok) {
      const data = await res.json();
      config.value.customSound = data.customSound;
      uploadedFileName.value = data.originalName || data.customSound;
      soundMode.value = 'custom';
      alert(`自定义音效上传成功：${uploadedFileName.value}`);
    } else {
      const text = await res.text();
      alert(`上传失败：${text || '服务器错误'}`);
    }
  } catch (err) {
    alert('上传音频失败，网络异常');
  } finally {
    uploadingSound.value = false;
    if (target) target.value = '';
  }
}

function stopPreview() {
  if (previewAudio) {
    previewAudio.pause();
    previewAudio.currentTime = 0;
    previewAudio = null;
  }
  isPlayingPreview.value = false;
}

function previewSound() {
  if (isPlayingPreview.value) {
    stopPreview();
    return;
  }

  let soundUrl = '/alert.mp3';
  if (soundMode.value === 'preset' && config.value.customSound) {
    soundUrl = `/sounds/${config.value.customSound}`;
  } else if (soundMode.value === 'custom' && config.value.customSound) {
    const custom = config.value.customSound.trim();
    if (custom.startsWith('http://') || custom.startsWith('https://')) {
      soundUrl = custom;
    } else {
      soundUrl = `/api/sound/current?t=${Date.now()}`;
    }
  } else {
    soundUrl = `/api/sound/current?t=${Date.now()}`;
  }

  stopPreview();
  try {
    previewAudio = new Audio(soundUrl);
    isPlayingPreview.value = true;

    previewAudio.onended = () => {
      isPlayingPreview.value = false;
      previewAudio = null;
    };

    previewAudio.onerror = () => {
      isPlayingPreview.value = false;
      previewAudio = null;
      alert('音效试听失败，可能音频格式不支持或链接失效');
    };

    previewAudio.play().catch(() => {
      isPlayingPreview.value = false;
      previewAudio = null;
    });
  } catch (e) {
    isPlayingPreview.value = false;
    previewAudio = null;
  }
}

function renderQRCode() {
  nextTick(() => {
    if (!qrCanvasRef.value) return;
    const url = frpPublicUrl.value || `http://${lanIPs.value[0] || '127.0.0.1'}:${config.value.port}`;
    QRCode.toCanvas(qrCanvasRef.value, url, {
      width: 192,
      margin: 1,
      color: { dark: '#0f172a', light: '#ffffff' }
    });
  });
}

async function saveConfig() {
  saving.value = true;
  try {
    const res = await fetch('/api/config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config.value)
    });
    if (res.ok) {
      alert('配置已成功保存！');
      renderQRCode();
    } else {
      alert('保存失败');
    }
  } catch (e) {
    alert('保存失败：网络异常');
  } finally {
    saving.value = false;
  }
}

function copyUrl(text: string) {
  navigator.clipboard.writeText(text);
  alert('已复制到剪贴板：' + text);
}
</script>

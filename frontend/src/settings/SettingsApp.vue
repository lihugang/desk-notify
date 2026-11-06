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
          <div v-for="(ip, idx) in lanIPs" :key="idx" class="flex items-center justify-between text-xs py-1">
            <span class="text-slate-600 font-mono">http://{{ ip }}:{{ config.port }}</span>
            <button @click="copyUrl(`http://${ip}:${config.port}`)" class="text-blue-600 hover:underline">复制链接</button>
          </div>
          <div v-if="frpPublicUrl" class="flex items-center justify-between text-xs py-1 border-t border-slate-200/60 pt-1.5">
            <span class="text-emerald-600 font-medium font-mono truncate mr-2">公网: {{ frpPublicUrl }}</span>
            <button @click="copyUrl(frpPublicUrl)" class="text-blue-600 hover:underline flex-shrink-0">复制链接</button>
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
import { QrCode, Settings } from 'lucide-vue-next';
import QRCode from 'qrcode';
import type { AppConfig } from '../types';

const activeTab = ref<'qrcode' | 'config'>('qrcode');
const qrCanvasRef = ref<HTMLCanvasElement | null>(null);
const lanIPs = ref<string[]>(['127.0.0.1']);
const frpPublicUrl = ref('');
const saving = ref(false);

const config = ref<AppConfig>({
  className: '高一(3)班',
  port: 8920,
  publicPassword: '',
  lanAuthRequired: false,
  defaultDuration: 30,
  soundEnabled: true,
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
    }
  } catch (e) {
    console.warn('获取配置失败', e);
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

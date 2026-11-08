<template>
  <div class="min-h-screen bg-slate-100 text-slate-800 p-4 sm:p-6 flex flex-col justify-between">
    <div class="max-w-2xl mx-auto w-full bg-white rounded-2xl shadow-sm border border-slate-200 overflow-hidden flex flex-col">
      <!-- Top Window Title & Close Bar -->
      <div class="px-5 py-3 bg-slate-50 border-b border-slate-200 flex items-center justify-between select-none">
        <div class="flex items-center gap-2">
          <div class="w-3 h-3 rounded-full bg-blue-500"></div>
          <span class="text-xs font-bold text-slate-700 tracking-wide">传纸条 · 班级设置与扫码</span>
          <span class="px-2 py-0.5 rounded-full bg-blue-100 text-blue-700 text-[10px] font-semibold">
            {{ config.className || '2907' }}
          </span>
        </div>
        <button
          @click="closeWindow"
          class="px-2.5 py-1 text-xs text-slate-500 hover:text-slate-800 hover:bg-slate-200/80 rounded-lg transition flex items-center gap-1"
          title="关闭本窗口"
        >
          <X class="w-3.5 h-3.5" />
          <span>关闭窗口</span>
        </button>
      </div>

      <!-- First Run Welcome Banner -->
      <div v-if="isFirstRun" class="mx-5 sm:mx-6 mt-4 p-3.5 bg-gradient-to-r from-blue-50 to-indigo-50 border border-blue-200 rounded-xl flex items-center justify-between text-xs text-blue-900 shadow-2xs gap-3">
        <div class="flex items-center gap-2.5">
          <Sparkles class="w-4 h-4 text-blue-600 flex-shrink-0" />
          <div>
            <span class="font-bold">欢迎使用 DeskNotify 传纸条！</span>
            <span class="text-blue-700 ml-1">首次打开已自动固定到底部任务栏并展开投递二维码。确认班级名称后关闭窗口，软件将常驻右下角托盘静默运行。</span>
          </div>
        </div>
        <div class="flex items-center gap-2 flex-shrink-0">
          <button
            type="button"
            @click="pinToTaskbar"
            :disabled="pinningTaskbar"
            class="px-2.5 py-1 text-xs bg-white hover:bg-blue-50 text-blue-700 border border-blue-200 rounded-lg font-medium transition shadow-2xs flex items-center gap-1 cursor-pointer disabled:opacity-50"
            title="固定到任务栏"
          >
            <Pin class="w-3.5 h-3.5 text-blue-600" />
            <span>{{ pinningTaskbar ? '正在固定...' : '固定到任务栏' }}</span>
          </button>
          <button
            @click="isFirstRun = false"
            class="text-blue-400 hover:text-blue-600 p-1 rounded-md transition"
            title="关闭提示"
          >
            <X class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      <!-- Main Navigation Tabs -->
      <div class="flex border-b border-slate-200 bg-white">
        <button
          @click="activeTab = 'qrcode'"
          :class="activeTab === 'qrcode' ? 'border-b-2 border-blue-600 text-blue-600 font-bold bg-blue-50/40' : 'text-slate-500 hover:text-slate-700 hover:bg-slate-50'"
          class="flex-1 py-3 text-xs sm:text-sm font-semibold text-center transition flex items-center justify-center gap-1.5"
        >
          <QrCode class="w-4 h-4" />
          <span>扫码投递 (手机端)</span>
        </button>

        <button
          @click="activeTab = 'basic'"
          :class="activeTab === 'basic' ? 'border-b-2 border-blue-600 text-blue-600 font-bold bg-blue-50/40' : 'text-slate-500 hover:text-slate-700 hover:bg-slate-50'"
          class="flex-1 py-3 text-xs sm:text-sm font-semibold text-center transition flex items-center justify-center gap-1.5"
        >
          <Settings class="w-4 h-4" />
          <span>基础设置</span>
        </button>

        <button
          @click="activeTab = 'security'"
          :class="activeTab === 'security' ? 'border-b-2 border-blue-600 text-blue-600 font-bold bg-blue-50/40' : 'text-slate-500 hover:text-slate-700 hover:bg-slate-50'"
          class="flex-1 py-3 text-xs sm:text-sm font-semibold text-center transition flex items-center justify-center gap-1.5"
        >
          <component :is="isUnlocked ? Unlock : Lock" class="w-4 h-4" :class="isUnlocked ? 'text-emerald-600' : 'text-slate-400'" />
          <span>安全与公网</span>
          <span v-if="!isUnlocked" class="text-[10px] px-1.5 py-0.2 bg-slate-100 text-slate-500 rounded">需口令</span>
          <span v-else class="text-[10px] px-1.5 py-0.2 bg-emerald-100 text-emerald-700 rounded font-normal">已解锁</span>
        </button>
      </div>

      <!-- Tab 1: QR Codes & Connection Addresses -->
      <div v-if="activeTab === 'qrcode'" class="p-5 sm:p-6 space-y-5">
        <div class="text-center">
          <h2 class="text-base font-bold text-slate-900">{{ config.className || '2907' }} · 教师投递入口</h2>
          <p class="text-xs text-slate-500 mt-1">老师使用手机微信或自带浏览器扫描下方对应地址的二维码，即可投递通知与课件</p>
        </div>

        <!-- Address Cards Grid -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <span class="text-xs font-bold text-slate-500 uppercase tracking-wider">可选连接地址列表</span>
            <span class="text-[11px] text-slate-400">点击二维码可放大全屏扫描</span>
          </div>

          <div
            v-for="item in displayAddresses"
            :key="item.id"
            class="bg-slate-50 hover:bg-slate-100/70 border border-slate-200 rounded-xl p-3.5 transition flex flex-col sm:flex-row items-center gap-4 group"
          >
            <!-- Rendered QR Code for THIS address -->
            <div
              class="bg-white p-2 border border-slate-200 rounded-xl shadow-2xs flex-shrink-0 cursor-pointer hover:ring-2 hover:ring-blue-400 transition"
              @click="openZoom(item)"
              title="点击放大二维码"
            >
              <canvas :ref="(el) => setQrCanvas(item.url, el)" class="w-28 h-28 rounded-lg block"></canvas>
            </div>

            <!-- Address Info & Actions -->
            <div class="flex-1 min-w-0 text-center sm:text-left space-y-1.5">
              <div class="flex items-center justify-center sm:justify-start gap-2 flex-wrap">
                <span class="text-xs font-bold text-slate-800">{{ item.type }}</span>
                <span class="px-2 py-0.5 text-[10px] rounded-full font-semibold border" :class="item.badgeClass">
                  {{ item.badge }}
                </span>
              </div>

              <div class="bg-white px-2.5 py-1.5 rounded-lg border border-slate-200 font-mono text-xs text-slate-700 select-all break-all">
                {{ item.url }}
              </div>

              <p class="text-[11px] text-slate-400 leading-normal">{{ item.desc }}</p>

              <div class="pt-1 flex items-center justify-center sm:justify-start gap-2">
                <button
                  @click="copyUrl(item.url)"
                  class="px-2.5 py-1 text-xs bg-white hover:bg-blue-50 text-blue-600 hover:text-blue-700 border border-slate-200 hover:border-blue-300 rounded-lg font-medium transition flex items-center gap-1 shadow-2xs"
                >
                  <Copy class="w-3 h-3" />
                  <span>复制链接</span>
                </button>
                <button
                  @click="openZoom(item)"
                  class="px-2.5 py-1 text-xs bg-white hover:bg-slate-100 text-slate-600 border border-slate-200 rounded-lg font-medium transition flex items-center gap-1 shadow-2xs"
                >
                  <QrCode class="w-3 h-3" />
                  <span>放大二维码</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Root CA Certificate Info & Download -->
        <div class="bg-indigo-50/70 border border-indigo-100 rounded-xl p-3.5 flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs">
          <div class="space-y-0.5">
            <div class="flex items-center gap-2">
              <p class="font-semibold text-indigo-900">传纸条根证书 (HTTPS)</p>
              <span
                class="px-2 py-0.5 rounded-full text-[10px] font-medium"
                :class="caInstalled ? 'bg-emerald-100 text-emerald-700 border border-emerald-200' : 'bg-amber-100 text-amber-700 border border-amber-200'"
              >
                {{ caInstalled ? '✓ 本机已受信任' : '未受信任' }}
              </span>
            </div>
            <p class="text-[11px] text-indigo-600/80">软件启动时会自动检查并信任根证书。如需在外接手机或平板安装证书，请点击右侧下载</p>
          </div>
          <div class="flex items-center gap-2 flex-shrink-0 flex-wrap">
            <a
              href="/DeskNotify-CA.cer"
              download="DeskNotify-CA.cer"
              class="px-3 py-1.5 bg-white hover:bg-indigo-50 border border-indigo-200 rounded-lg text-indigo-700 text-xs font-semibold shadow-2xs transition inline-flex items-center gap-1.5"
            >
              <Download class="w-3.5 h-3.5" />
              <span>下载根证书 (.cer)</span>
            </a>
          </div>
        </div>
      </div>

      <!-- Tab 2: Level 1 Basic Configuration -->
      <div v-if="activeTab === 'basic'" class="p-5 sm:p-6 space-y-4">
        <div>
          <label class="block text-xs font-semibold text-slate-700 mb-1">班级名称</label>
          <input
            v-model="config.className"
            type="text"
            class="w-full px-3 py-2 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
            placeholder="如：2907"
          />
          <p class="text-[11px] text-slate-400 mt-1">显示在通知窗口头部及教师投递页面标题</p>
        </div>

        <!-- Notification Font Size Adjustment Section -->
        <div class="pt-2 border-t border-slate-100 space-y-3">
          <div class="flex items-center justify-between">
            <div>
              <span class="text-xs font-semibold text-slate-700">通知卡片正文字号</span>
              <p class="text-[11px] text-slate-400 mt-0.5">调整教室大屏通知卡片的文字大小，方便远距离与后排同学清晰阅读</p>
            </div>
            <span class="px-2.5 py-0.5 rounded-full bg-blue-50 text-blue-700 border border-blue-200 font-mono text-xs font-semibold">
              {{ currentFontSizeNumber }}px
            </span>
          </div>

          <div class="space-y-3 bg-slate-50 p-3.5 rounded-xl border border-slate-200">
            <!-- Preset Buttons -->
            <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
              <button
                v-for="item in fontSizePresets"
                :key="item.value"
                type="button"
                @click="setFontSize(item.value)"
                :class="currentFontSizeNumber === item.size ? 'bg-blue-50 border-blue-500 text-blue-700 font-semibold ring-1 ring-blue-400 shadow-2xs' : 'bg-white border-slate-200 text-slate-600 hover:bg-slate-100/80'"
                class="py-2 px-2.5 text-xs rounded-xl border text-center transition flex flex-col items-center justify-center gap-0.5 cursor-pointer"
              >
                <span class="font-bold">{{ item.label }}</span>
                <span class="text-[10px] text-slate-400 font-normal">{{ item.desc }} ({{ item.size }}px)</span>
              </button>
            </div>

            <!-- Fine-tuning Slider & Quick Steppers -->
            <div class="bg-white p-3 rounded-lg border border-slate-200/80 space-y-2">
              <div class="flex items-center justify-between text-xs">
                <span class="text-slate-500 font-medium flex items-center gap-1.5">
                  <Type class="w-3.5 h-3.5 text-blue-600" />
                  <span>精细滑块调节 (14px - 32px)</span>
                </span>
                <div class="flex items-center gap-1">
                  <button
                    type="button"
                    @click="stepFontSize(-1)"
                    :disabled="currentFontSizeNumber <= 14"
                    class="w-6 h-6 rounded flex items-center justify-center bg-slate-100 hover:bg-slate-200 text-slate-700 disabled:opacity-40 transition text-xs font-bold cursor-pointer"
                    title="缩小 1px"
                  >
                    -
                  </button>
                  <span class="font-mono font-semibold text-slate-800 text-xs px-2">{{ currentFontSizeNumber }}px</span>
                  <button
                    type="button"
                    @click="stepFontSize(1)"
                    :disabled="currentFontSizeNumber >= 32"
                    class="w-6 h-6 rounded flex items-center justify-center bg-slate-100 hover:bg-slate-200 text-slate-700 disabled:opacity-40 transition text-xs font-bold cursor-pointer"
                    title="放大 1px"
                  >
                    +
                  </button>
                </div>
              </div>
              <input
                type="range"
                min="14"
                max="32"
                step="1"
                :value="currentFontSizeNumber"
                @input="handleSliderChange"
                class="w-full accent-blue-600 cursor-pointer"
              />
            </div>

            <!-- Live Preview Box -->
            <div class="bg-white p-3.5 rounded-lg border border-slate-200/80 space-y-1.5">
              <div class="flex items-center justify-between text-[11px] text-slate-400 pb-1 border-b border-slate-100">
                <span class="font-medium text-slate-500 flex items-center gap-1">
                  <Eye class="w-3 h-3 text-slate-400" />
                  <span>实时效果预览</span>
                </span>
                <span>当前应用：{{ currentFontSizeNumber }}px</span>
              </div>
              <div
                class="markdown-content text-slate-800 leading-relaxed transition-all"
                :style="{ fontSize: currentFontSizeNumber + 'px' }"
              >
                <p><strong>【班级通知】</strong> 今天下午第三节课在多媒体教室开会，请各组组长带好笔记本和课件准时参加。</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Notification Sound Settings Section -->
        <div class="pt-2 border-t border-slate-100 space-y-3">
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
                class="inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-lg transition shadow-2xs"
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
                内置精选风格
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
                <span class="px-2 py-0.5 bg-blue-50 text-blue-600 rounded text-[10px] font-medium">开箱即用</span>
              </div>
              <p class="text-[11px] text-slate-400">已完整内嵌于单一可执行程序中，绿色免安装。</p>
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
                  :class="config.customSound === preset.id ? 'bg-blue-50 border-blue-500 text-blue-700 font-semibold ring-1 ring-blue-400 shadow-2xs' : 'bg-slate-50 border-slate-200 text-slate-600 hover:bg-slate-100'"
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
                    class="px-3 py-1.5 bg-slate-100 hover:bg-slate-200 text-slate-700 rounded-lg text-xs font-medium border border-slate-200 transition flex items-center gap-1.5 shadow-2xs"
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

        <!-- Notification Archive Directory Section -->
        <div class="pt-3 border-t border-slate-100 space-y-2">
          <div class="flex items-center justify-between">
            <div>
              <span class="text-xs font-semibold text-slate-700">通知与课件存档目录</span>
              <p class="text-[11px] text-slate-400 mt-0.5">接收到的通知富文本、图片与课件文件会自动以只读硬链接保存在此文件夹中</p>
            </div>
          </div>

          <div class="bg-slate-50 p-3 rounded-xl border border-slate-200 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div class="flex items-center gap-2 flex-1 min-w-0">
              <Folder class="w-4 h-4 text-blue-600 flex-shrink-0" />
              <span class="text-xs font-mono text-slate-600 truncate select-all" :title="config.documentsDir">
                {{ config.documentsDir || ('我的文档\\DeskNotify\\' + (config.className || '班级通知')) }}
              </span>
            </div>
            <div class="flex items-center gap-2 flex-shrink-0">
              <button
                type="button"
                @click="openArchiveDir"
                :disabled="openingDir"
                class="px-2.5 py-1.5 text-xs bg-white hover:bg-blue-50 text-blue-600 border border-slate-200 hover:border-blue-300 rounded-lg font-medium transition flex items-center gap-1 shadow-2xs cursor-pointer disabled:opacity-60"
                title="在资源管理器中打开此文件夹"
              >
                <FolderOpen class="w-3.5 h-3.5" />
                <span>{{ openingDir ? '正在呼出...' : '打开通知存档目录' }}</span>
              </button>
              <button
                type="button"
                @click="copyArchiveDir"
                class="px-2.5 py-1.5 text-xs bg-white hover:bg-slate-100 text-slate-600 border border-slate-200 rounded-lg font-medium transition flex items-center gap-1 shadow-2xs cursor-pointer"
                title="复制目录路径"
              >
                <Copy class="w-3.5 h-3.5" />
                <span>复制路径</span>
              </button>
            </div>
          </div>
        </div>

        <!-- System Logs Directory Section -->
        <div class="pt-3 border-t border-slate-100 space-y-2">
          <div class="flex items-center justify-between">
            <div>
              <span class="text-xs font-semibold text-slate-700">运行日志目录</span>
              <p class="text-[11px] text-slate-400 mt-0.5">系统按天自动记录运行、网络穿透与故障排查日志（位于 %APPDATA%/DeskNotify/logs）</p>
            </div>
          </div>

          <div class="bg-slate-50 p-3 rounded-xl border border-slate-200 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div class="flex items-center gap-2 flex-1 min-w-0">
              <component :is="FileText" class="w-4 h-4 text-emerald-600 flex-shrink-0" />
              <span class="text-xs font-mono text-slate-600 truncate select-all" :title="config.logsDir">
                {{ config.logsDir || '%APPDATA%\\DeskNotify\\logs' }}
              </span>
            </div>
            <div class="flex items-center gap-2 flex-shrink-0">
              <button
                type="button"
                @click="openLogsDir"
                :disabled="openingLogsDir"
                class="px-2.5 py-1.5 text-xs bg-white hover:bg-emerald-50 text-emerald-600 border border-slate-200 hover:border-emerald-300 rounded-lg font-medium transition flex items-center gap-1 shadow-2xs cursor-pointer disabled:opacity-60"
                title="在资源管理器中打开日志文件夹"
              >
                <FolderOpen class="w-3.5 h-3.5" />
                <span>{{ openingLogsDir ? '正在呼出...' : '打开日志目录' }}</span>
              </button>
              <button
                type="button"
                @click="copyLogsDir"
                class="px-2.5 py-1.5 text-xs bg-white hover:bg-slate-100 text-slate-600 border border-slate-200 rounded-lg font-medium transition flex items-center gap-1 shadow-2xs cursor-pointer"
                title="复制日志目录路径"
              >
                <Copy class="w-3.5 h-3.5" />
                <span>复制路径</span>
              </button>
            </div>
          </div>
        </div>

        <!-- Shortcuts & Pinning Section -->
        <div class="pt-3 border-t border-slate-100 space-y-2">
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div>
              <span class="text-xs font-semibold text-slate-700">桌面与任务栏快捷方式</span>
              <p class="text-[11px] text-slate-400 mt-0.5">为方便快速唤出传纸条，可将应用添加到桌面或固定到底部任务栏</p>
            </div>
            <div class="flex items-center gap-2 flex-shrink-0 flex-wrap">
              <button
                type="button"
                @click="createDesktopShortcut"
                :disabled="creatingShortcut"
                class="px-3 py-1.5 text-xs bg-white hover:bg-blue-50 text-blue-700 border border-slate-200 hover:border-blue-300 rounded-lg font-medium transition flex items-center gap-1.5 shadow-2xs cursor-pointer disabled:opacity-50"
              >
                <Monitor class="w-3.5 h-3.5 text-blue-600" />
                <span>{{ creatingShortcut ? '正在创建...' : '添加到桌面快捷方式' }}</span>
              </button>
              <button
                type="button"
                @click="pinToTaskbar"
                :disabled="pinningTaskbar"
                class="px-3 py-1.5 text-xs bg-white hover:bg-blue-50 text-blue-700 border border-slate-200 hover:border-blue-300 rounded-lg font-medium transition flex items-center gap-1.5 shadow-2xs cursor-pointer disabled:opacity-50"
              >
                <Pin class="w-3.5 h-3.5 text-blue-600" />
                <span>{{ pinningTaskbar ? '正在固定...' : '固定到底部任务栏' }}</span>
              </button>
            </div>
          </div>
        </div>

        <div class="pt-3 flex justify-end">
          <button
            @click="saveBasicConfig"
            :disabled="saving"
            class="px-6 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-xl text-sm font-semibold shadow-xs disabled:opacity-50 transition"
          >
            {{ saving ? '保存中...' : '保存基础设置' }}
          </button>
        </div>
      </div>

      <!-- Tab 3: Level 2 Security & Public FRP Configuration -->
      <div v-if="activeTab === 'security'" class="p-5 sm:p-6 space-y-5">
        <!-- Auth Gate: If NOT unlocked, show password prompt -->
        <div v-if="!isUnlocked" class="p-6 bg-slate-50 rounded-2xl border border-slate-200 text-center space-y-4 max-w-md mx-auto">
          <div class="w-12 h-12 bg-blue-100 text-blue-600 rounded-2xl mx-auto flex items-center justify-center">
            <Lock class="w-6 h-6" />
          </div>
          <div>
            <h3 class="text-sm font-bold text-slate-800">安全验证 · 统一密码</h3>
            <p class="text-xs text-slate-500 mt-1">修改公网服务与系统安全设置属于高级操作，请输入统一密码以解锁配置</p>
          </div>

          <div class="bg-amber-50 border border-amber-200 rounded-xl p-2.5 text-xs text-amber-800 text-left">
            <span>初始默认密码：<strong class="font-mono text-amber-900">123456</strong></span>
            <span class="block text-[10px] text-amber-700 mt-0.5">投递密码与管理密码已统一为同一口令，采用 bcrypt 加盐安全比对</span>
          </div>

          <div class="space-y-2">
            <div class="relative">
              <input
                v-model="adminPasswordInput"
                :type="showAdminPassword ? 'text' : 'password'"
                @keyup.enter="unlockAdmin"
                class="w-full px-3 py-2 pr-10 border border-slate-300 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
                placeholder="请输入系统统一密码"
              />
              <button
                type="button"
                @click="showAdminPassword = !showAdminPassword"
                class="absolute right-3 top-2.5 text-slate-400 hover:text-slate-600"
              >
                <component :is="showAdminPassword ? EyeOff : Eye" class="w-4 h-4" />
              </button>
            </div>
            <p v-if="adminError" class="text-xs text-red-600 text-left">{{ adminError }}</p>
          </div>

          <button
            @click="unlockAdmin"
            :disabled="verifyingAdmin"
            class="w-full py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-xl text-xs font-semibold shadow-xs disabled:opacity-50 transition"
          >
            {{ verifyingAdmin ? '正在验证...' : '验证密码并解锁配置' }}
          </button>
        </div>

        <!-- Unlocked View: Level 2 Settings -->
        <div v-else class="space-y-5">
          <div class="flex items-center justify-between bg-emerald-50 border border-emerald-200 p-3 rounded-xl">
            <div class="flex items-center gap-2">
              <ShieldCheck class="w-4 h-4 text-emerald-600" />
              <span class="text-xs font-semibold text-emerald-800">已解锁高级配置权限</span>
            </div>
            <button
              @click="relockAdmin"
              class="px-2.5 py-1 text-xs bg-white text-slate-600 hover:text-slate-900 border border-emerald-200 rounded-lg shadow-2xs transition"
            >
              重新锁定
            </button>
          </div>

          <!-- Section 1: Unified System Password -->
          <div class="space-y-2 bg-slate-50 p-4 rounded-xl border border-slate-200">
            <div class="flex items-center justify-between">
              <label class="block text-xs font-semibold text-slate-800">系统统一密码 (投递与管理口令)</label>
              <span class="text-[10px] text-blue-600 bg-blue-50 px-2 py-0.5 rounded-full font-medium">投递与管理合一</span>
            </div>
            <input
              v-model="newPassword"
              type="password"
              class="w-full px-3 py-2 bg-white border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              :placeholder="(config.hasPassword || config.hasPublicPassword) ? '留空表示保持原有密码不变' : '设置系统统一密码'"
            />
            <p class="text-[11px] text-slate-400 leading-relaxed">
              投递密码与管理密码已统一合并为同一个密码。外部网络教师扫码投递时验证此密码，同时用于解锁与修改本机设置。修改后两处同时生效，绝不保存明文。
            </p>
          </div>

          <!-- Section 2: LAN Authentication Required -->
          <div class="space-y-3 bg-slate-50 p-4 rounded-xl border border-slate-200">
            <div class="flex items-center justify-between">
              <div>
                <span class="text-xs font-semibold text-slate-800">局域网访问强制验证</span>
                <p class="text-[11px] text-slate-400 mt-0.5">
                  开启后，校园局域网内的教师扫码投递也必须输入投递口令；关闭时局域网内直接免密投递
                </p>
              </div>
              <input
                v-model="config.lanAuthRequired"
                type="checkbox"
                class="w-4 h-4 rounded text-blue-600 cursor-pointer"
              />
            </div>
          </div>

          <!-- Section 3: Expose Service to Public Network (Built-in FreeFRP / OpenFrp) -->
          <div class="space-y-3 bg-slate-50 p-4 rounded-xl border border-slate-200">
            <div class="flex items-center justify-between">
              <div>
                <div class="flex items-center gap-2">
                  <span class="text-xs font-semibold text-slate-800">暴露服务到公网 (FRP / OpenFrp)</span>
                  <button
                    v-if="config.frp.enabled"
                    @click="showFrpImportModal = true"
                    class="px-2 py-0.5 bg-blue-50 hover:bg-blue-100 text-blue-600 rounded text-[11px] font-medium border border-blue-200/60 transition flex items-center gap-1 cursor-pointer"
                  >
                    <Download class="w-3 h-3" />
                    导入配置 (TOML/INI)
                  </button>
                  <button
                    v-if="config.frp.enabled"
                    @click="resetToFreeFRP"
                    class="px-2 py-0.5 bg-slate-100 hover:bg-slate-200 text-slate-600 rounded text-[11px] font-medium border border-slate-200/60 transition flex items-center gap-1 cursor-pointer"
                    title="重置为默认 FreeFRP 公共节点配置"
                  >
                    <RotateCcw class="w-3 h-3" />
                    恢复 FreeFRP 默认
                  </button>
                </div>
                <p class="text-[11px] text-slate-400 mt-0.5">支持内置 FreeFRP 或导入 OpenFrp 节点，开启后校外教师可在任意网络扫码投递</p>
              </div>
              <input v-model="config.frp.enabled" type="checkbox" class="w-4 h-4 rounded text-blue-600 cursor-pointer" />
            </div>

            <div v-if="config.frp.enabled" class="space-y-3 bg-white p-3.5 rounded-xl border border-slate-200/80 mt-2">
              <div class="grid grid-cols-3 gap-2">
                <div class="col-span-2">
                  <label class="block text-[11px] font-medium text-slate-500 mb-1">FRP 服务器地址 (Server Addr)</label>
                  <input
                    v-model="config.frp.serverAddr"
                    type="text"
                    placeholder="如 cn-hk-bgp-4.ofalias.net 或 frp.freefrp.net"
                    class="w-full px-2.5 py-1.5 bg-slate-50 border border-slate-200 rounded text-xs focus:ring-1 focus:ring-blue-500 focus:bg-white"
                  />
                </div>
                <div>
                  <label class="block text-[11px] font-medium text-slate-500 mb-1">服务器端口 (Port)</label>
                  <input
                    v-model.number="config.frp.serverPort"
                    type="number"
                    placeholder="7000 或 8120"
                    class="w-full px-2.5 py-1.5 bg-slate-50 border border-slate-200 rounded text-xs focus:ring-1 focus:ring-blue-500 focus:bg-white"
                  />
                </div>
              </div>

              <div class="grid grid-cols-2 gap-2">
                <div>
                  <label class="block text-[11px] font-medium text-slate-500 mb-1">用户标识 User (OpenFrp 专属，普通 FRP 留空)</label>
                  <input
                    v-model="config.frp.user"
                    type="text"
                    placeholder="OpenFrp 用户标识 (user)"
                    class="w-full px-2.5 py-1.5 bg-slate-50 border border-slate-200 rounded text-xs focus:ring-1 focus:ring-blue-500 focus:bg-white font-mono"
                  />
                </div>
                <div>
                  <label class="block text-[11px] font-medium text-slate-500 mb-1">通信密钥 Token</label>
                  <input
                    v-model="config.frp.token"
                    type="password"
                    placeholder="认证 Token"
                    class="w-full px-2.5 py-1.5 bg-slate-50 border border-slate-200 rounded text-xs focus:ring-1 focus:ring-blue-500 focus:bg-white"
                  />
                </div>
              </div>

              <div class="grid grid-cols-2 gap-2">
                <div>
                  <label class="block text-[11px] font-medium text-slate-500 mb-1">隧道名称 (Proxy Name)</label>
                  <input
                    v-model="config.frp.proxyName"
                    type="text"
                    placeholder="如 test (留空自动生成)"
                    class="w-full px-2.5 py-1.5 bg-slate-50 border border-slate-200 rounded text-xs focus:ring-1 focus:ring-blue-500 focus:bg-white font-mono"
                  />
                </div>
                <div>
                  <label class="block text-[11px] font-medium text-slate-500 mb-1">远程绑定端口 (Remote Port)</label>
                  <input
                    v-model.number="config.frp.remotePort"
                    type="number"
                    placeholder="如 50847"
                    class="w-full px-2.5 py-1.5 bg-slate-50 border border-slate-200 rounded text-xs focus:ring-1 focus:ring-blue-500 focus:bg-white font-mono"
                  />
                </div>
              </div>

              <!-- TLS Options -->
              <div class="p-2.5 bg-slate-50/80 rounded-lg border border-slate-200/60 space-y-2">
                <div class="flex items-center justify-between">
                  <div>
                    <span class="text-[11px] font-medium text-slate-700">传输加密 TLS (transport.tls.enable)</span>
                    <p class="text-[10px] text-slate-400">OpenFrp 节点建议勾选；内置 FreeFRP 通常保持关闭</p>
                  </div>
                  <input
                    v-model="config.frp.tlsEnable"
                    type="checkbox"
                    class="w-3.5 h-3.5 rounded text-blue-600 cursor-pointer"
                  />
                </div>

                <div v-if="config.frp.tlsEnable" class="flex items-center justify-between pt-1 border-t border-slate-200/40">
                  <div>
                    <span class="text-[11px] font-medium text-slate-700">禁用自定义首字节 (disableCustomTLSFirstByte)</span>
                    <p class="text-[10px] text-slate-400">OpenFrp 节点建议关闭 (false)；标准原版节点若连不上可尝试开启 (true)</p>
                  </div>
                  <input
                    v-model="config.frp.disableCustomTLSFirstByte"
                    type="checkbox"
                    class="w-3.5 h-3.5 rounded text-blue-600 cursor-pointer"
                  />
                </div>
              </div>

              <div class="text-[11px] text-slate-500 bg-slate-50 p-2 rounded border border-slate-200/60 flex items-center justify-between">
                <div>
                  <span>外网访问地址：</span>
                  <span class="font-mono text-blue-600 font-semibold">{{ frpPublicUrl || ('http://' + (config.frp.serverAddr || 'frp.freefrp.net') + ':' + config.frp.remotePort) }}</span>
                </div>
                <span v-if="frpStatus" :class="frpStatus === 'connected' ? 'text-emerald-600 font-medium' : 'text-slate-400'">
                  {{ frpStatus === 'connected' ? '● 已连接' : frpStatus === 'connecting' ? '◌ 连接中...' : '○ 未连接' }}
                </span>
              </div>
            </div>
          </div>

          <div class="pt-3 flex justify-end">
            <button
              @click="saveSecurityConfig"
              :disabled="saving"
              class="px-6 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-xl text-sm font-semibold shadow-xs disabled:opacity-50 transition"
            >
              {{ saving ? '保存中...' : '保存安全与公网配置' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Zoom QR Modal -->
    <div
      v-if="zoomedAddress"
      class="fixed inset-0 bg-slate-900/60 backdrop-blur-xs flex items-center justify-center p-4 z-50 animate-fade-in"
      @click.self="zoomedAddress = null"
    >
      <div class="bg-white rounded-2xl max-w-sm w-full p-6 text-center space-y-4 shadow-2xl border border-slate-100">
        <div class="flex items-center justify-between">
          <span class="text-xs font-bold text-slate-700">{{ zoomedAddress.type }}</span>
          <button @click="zoomedAddress = null" class="p-1 hover:bg-slate-100 rounded-lg text-slate-400 hover:text-slate-700">
            <X class="w-4 h-4" />
          </button>
        </div>

        <div class="flex justify-center p-3 bg-white border border-slate-200 rounded-2xl shadow-xs inline-block mx-auto">
          <canvas ref="zoomCanvasRef" class="w-56 h-56 rounded-lg block"></canvas>
        </div>

        <div class="bg-slate-50 p-2.5 rounded-lg border border-slate-200 font-mono text-xs text-slate-700 select-all break-all">
          {{ zoomedAddress.url }}
        </div>

        <div class="flex gap-2">
          <button
            @click="copyUrl(zoomedAddress.url)"
            class="flex-1 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-xl text-xs font-semibold shadow-xs transition flex items-center justify-center gap-1.5"
          >
            <Copy class="w-3.5 h-3.5" />
            <span>复制连接</span>
          </button>
          <button
            @click="zoomedAddress = null"
            class="px-4 py-2 bg-slate-100 hover:bg-slate-200 text-slate-700 rounded-xl text-xs font-semibold transition"
          >
            关闭
          </button>
        </div>
      </div>
    </div>

    <!-- FRP Import Modal -->
    <div
      v-if="showFrpImportModal"
      class="fixed inset-0 bg-slate-900/60 backdrop-blur-xs flex items-center justify-center p-4 z-50 animate-fade-in"
      @click.self="showFrpImportModal = false"
    >
      <div class="bg-white rounded-2xl max-w-lg w-full p-5 space-y-4 shadow-2xl border border-slate-100">
        <div class="flex items-center justify-between border-b border-slate-100 pb-3">
          <div class="flex items-center gap-2">
            <Download class="w-4 h-4 text-blue-600" />
            <h3 class="text-sm font-bold text-slate-800">一键导入 FRP / OpenFrp 配置</h3>
          </div>
          <button @click="showFrpImportModal = false" class="p-1 hover:bg-slate-100 rounded-lg text-slate-400 hover:text-slate-700 cursor-pointer">
            <X class="w-4 h-4" />
          </button>
        </div>

        <p class="text-xs text-slate-500 leading-relaxed">
          直接将从 OpenFrp 或其他平台复制的配置文件文本（支持 <code>frpc.toml</code> 或旧版 <code>frpc.ini</code>）粘贴在下方，系统将自动识别并回填各项参数：
        </p>

        <textarea
          v-model="frpImportText"
          rows="10"
          placeholder="例如：
serverAddr = 'cn-hk-bgp-4.ofalias.net'
serverPort = 8120
user = 'f58dbf7c...'

[auth]
token = '...'

[[proxies]]
name = 'test'
remotePort = 50847

[transport.tls]
disableCustomTLSFirstByte = false
enable = true"
          class="w-full p-3 bg-slate-50 border border-slate-200 rounded-xl text-xs font-mono focus:ring-2 focus:ring-blue-500 focus:bg-white resize-none"
        ></textarea>

        <div v-if="frpImportError" class="text-xs text-rose-500 bg-rose-50 p-2.5 rounded-lg border border-rose-100">
          {{ frpImportError }}
        </div>

        <div class="flex justify-end gap-2 pt-2">
          <button
            @click="showFrpImportModal = false"
            class="px-4 py-2 text-xs font-medium text-slate-600 hover:bg-slate-100 rounded-xl transition cursor-pointer"
          >
            取消
          </button>
          <button
            @click="applyFrpImport"
            class="px-5 py-2 text-xs font-semibold bg-blue-600 hover:bg-blue-700 text-white rounded-xl shadow-xs transition cursor-pointer"
          >
            解析并填入
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick, watch } from 'vue';
import {
  QrCode,
  Settings,
  Lock,
  Unlock,
  ShieldCheck,
  Volume2,
  VolumeX,
  Upload,
  RotateCcw,
  Copy,
  X,
  Eye,
  EyeOff,
  Sparkles,
  Folder,
  FolderOpen,
  FileText,
  Type,
  Pin,
  Monitor,
  Download
} from 'lucide-vue-next';
import QRCode from 'qrcode';
import type { AppConfig } from '../types';
import { parseFRPConfigText } from '../utils/frpParser';

interface AddressItem {
  id: string;
  type: string;
  badge: string;
  badgeClass: string;
  url: string;
  desc: string;
}

const activeTab = ref<'qrcode' | 'basic' | 'security'>('qrcode');
const lanIPs = ref<string[]>(['127.0.0.1']);
const frpPublicUrl = ref('');
const frpStatus = ref('');
const showFrpImportModal = ref(false);
const frpImportText = ref('');
const frpImportError = ref('');

function applyFrpImport() {
  frpImportError.value = '';
  if (!frpImportText.value.trim()) {
    frpImportError.value = '请粘贴配置文件内容 (支持 TOML / INI 格式)';
    return;
  }
  const parsed = parseFRPConfigText(frpImportText.value);
  if (!parsed.serverAddr && !parsed.token && !parsed.user && !parsed.remotePort) {
    frpImportError.value = '未能解析到有效的 FRP 配置项，请确认内容格式';
    return;
  }

  if (parsed.serverAddr) config.value.frp.serverAddr = parsed.serverAddr;
  if (parsed.serverPort) config.value.frp.serverPort = parsed.serverPort;
  if (parsed.user !== undefined) config.value.frp.user = parsed.user;
  if (parsed.token) config.value.frp.token = parsed.token;
  if (parsed.remotePort) config.value.frp.remotePort = parsed.remotePort;
  if (parsed.proxyName !== undefined) config.value.frp.proxyName = parsed.proxyName;
  if (parsed.protocol) config.value.frp.protocol = parsed.protocol;
  if (parsed.tlsEnable !== undefined) config.value.frp.tlsEnable = parsed.tlsEnable;
  if (parsed.disableCustomTLSFirstByte !== undefined) {
    config.value.frp.disableCustomTLSFirstByte = parsed.disableCustomTLSFirstByte;
  }
  config.value.frp.enabled = true;

  showFrpImportModal.value = false;
  frpImportText.value = '';
  frpImportError.value = '';
  alert('配置导入并解析成功！已自动填充表单各项参数，请点击“保存安全与公网配置”以生效。');
}

function resetToFreeFRP() {
  config.value.frp.serverAddr = 'frp.freefrp.net';
  config.value.frp.serverPort = 7000;
  config.value.frp.token = 'freefrp.net';
  config.value.frp.user = '';
  config.value.frp.proxyName = '';
  config.value.frp.protocol = 'tcp';
  config.value.frp.tlsEnable = false;
  config.value.frp.disableCustomTLSFirstByte = false;
  if (!config.value.frp.remotePort) {
    config.value.frp.remotePort = 10001 + Math.floor(Math.random() * 39999);
  }
  alert('已重置为默认 FreeFRP 节点配置，请点击“保存安全与公网配置”以应用。');
}
const saving = ref(false);
const isFirstRun = ref(false);

// Level 2 Security state
const isUnlocked = ref(false);
const adminPasswordInput = ref('');
const showAdminPassword = ref(false);
const verifyingAdmin = ref(false);
const adminError = ref('');
const adminToken = ref('');

const newPassword = ref('');

// Sound preview & upload
const audioFileInputRef = ref<HTMLInputElement | null>(null);
const uploadingSound = ref(false);
const uploadedFileName = ref('');
const soundMode = ref<'default' | 'preset' | 'custom'>('default');
const isPlayingPreview = ref(false);
let previewAudio: HTMLAudioElement | null = null;

// Zoom Modal
const zoomedAddress = ref<AddressItem | null>(null);
const zoomCanvasRef = ref<HTMLCanvasElement | null>(null);

const presets = [
  { id: 'glass.mp3', label: '清脆风铃', desc: '灵动通透' },
  { id: 'minimal.mp3', label: '极简提示', desc: '干脆利落' },
  { id: 'soft.mp3', label: '柔和气泡', desc: '温润悦耳' },
  { id: 'polite.mp3', label: '优雅轻响', desc: '经典现代' },
  { id: 'chime.mp3', label: '空灵旋律', desc: '舒缓悠扬' },
];

const fontSizePresets = [
  { value: '16px', size: 16, label: '标准', desc: '默认推荐' },
  { value: '18px', size: 18, label: '较大', desc: '中等教室' },
  { value: '22px', size: 22, label: '特大', desc: '宽敞大班' },
  { value: '26px', size: 26, label: '超大', desc: '阶梯教室' },
];

const config = ref<AppConfig>({
  className: '2907',
  port: 8920,
  hasPassword: true,
  hasPublicPassword: true,
  lanAuthRequired: false,
  soundEnabled: true,
  customSound: '',
  fontSize: '16px',
  autoStart: true,
  frp: {
    enabled: true,
    serverAddr: 'frp.freefrp.net',
    serverPort: 7000,
    token: 'freefrp.net',
    remotePort: 28932,
    user: '',
    proxyName: '',
    protocol: 'tcp',
    tlsEnable: false,
    disableCustomTLSFirstByte: false
  }
});

const currentFontSizeNumber = computed(() => {
  const raw = config.value.fontSize || '16px';
  const num = parseInt(raw, 10);
  return isNaN(num) ? 16 : num;
});

function notifyFontSizeChange(size: string) {
  try {
    localStorage.setItem('desknotify_font_size', size);
    if (typeof BroadcastChannel !== 'undefined') {
      const channel = new BroadcastChannel('desknotify_font_channel');
      channel.postMessage({ fontSize: size });
      channel.close();
    }
  } catch {}
}

function setFontSize(val: string) {
  config.value.fontSize = val;
  notifyFontSizeChange(val);
}

function stepFontSize(delta: number) {
  const next = Math.max(14, Math.min(32, currentFontSizeNumber.value + delta));
  setFontSize(`${next}px`);
}

function handleSliderChange(e: Event) {
  const val = Number((e.target as HTMLInputElement).value);
  if (!isNaN(val)) {
    setFontSize(`${val}px`);
  }
}

// Addresses computation following the 192 rule and FRP inclusion
const displayAddresses = computed<AddressItem[]>(() => {
  const list: AddressItem[] = [];
  const basePort = config.value.port || 8920;

  // 192 Rule: If any IP starts with 192., keep ONLY those starting with 192.; otherwise keep all
  const rawIPs = lanIPs.value && lanIPs.value.length > 0 ? lanIPs.value : ['127.0.0.1'];
  const has192 = rawIPs.some(ip => ip.startsWith('192.'));
  const effectiveIPs = has192 ? rawIPs.filter(ip => ip.startsWith('192.')) : rawIPs;

  for (const ip of effectiveIPs) {
    list.push({
      id: `http-${ip}`,
      type: '局域网 HTTP',
      badge: 'HTTP',
      badgeClass: 'bg-blue-50 text-blue-700 border-blue-200',
      url: `http://${ip}:${basePort}`,
      desc: '手机连接同一教室/校园 Wi-Fi 即可扫码直接投递'
    });
    list.push({
      id: `https-${ip}`,
      type: '局域网 HTTPS (推荐)',
      badge: 'HTTPS 加密',
      badgeClass: 'bg-indigo-50 text-indigo-700 border-indigo-200',
      url: `https://${ip}:${basePort + 1}`,
      desc: '加密通道，支持在手机上安装 PWA 桌面独立快捷方式'
    });
  }

  // Include FRP when enabled
  if (config.value.frp.enabled) {
    const publicUrl = frpPublicUrl.value || (config.value.frp.serverAddr && config.value.frp.remotePort ? `http://${config.value.frp.serverAddr}:${config.value.frp.remotePort}` : '');
    if (publicUrl) {
      list.push({
        id: 'frp-public',
        type: '暴露服务到公网',
        badge: '公网访问',
        badgeClass: 'bg-emerald-50 text-emerald-700 border-emerald-200',
        url: publicUrl,
        desc: '教师可在校外、家中或使用 4G/5G 手机网络直接扫码投递'
      });
    }
  }

  return list;
});

const qrCanvases = new Map<string, HTMLCanvasElement>();

function setQrCanvas(url: string, el: any) {
  if (el) {
    qrCanvases.set(url, el as HTMLCanvasElement);
    QRCode.toCanvas(el, url, {
      width: 112,
      margin: 1,
      color: { dark: '#0f172a', light: '#ffffff' }
    }).catch(() => {});
  }
}

watch(displayAddresses, () => {
  nextTick(() => {
    for (const item of displayAddresses.value) {
      const canvas = qrCanvases.get(item.url);
      if (canvas) {
        QRCode.toCanvas(canvas, item.url, {
          width: 112,
          margin: 1,
          color: { dark: '#0f172a', light: '#ffffff' }
        }).catch(() => {});
      }
    }
  });
}, { deep: true });

onMounted(async () => {
  window.addEventListener('beforeunload', () => {
    relockAdmin();
  });
  window.addEventListener('pagehide', () => {
    relockAdmin();
  });

  if (typeof window !== 'undefined') {
    (window as any).__dnRelockAdmin = () => {
      relockAdmin();
    };
  }

  await fetchStatusAndConfig();
  nextTick(() => {
    for (const item of displayAddresses.value) {
      const canvas = qrCanvases.get(item.url);
      if (canvas) {
        QRCode.toCanvas(canvas, item.url, {
          width: 112,
          margin: 1,
          color: { dark: '#0f172a', light: '#ffffff' }
        }).catch(() => {});
      }
    }
  });
});

async function fetchStatusAndConfig() {
  try {
    // If user already holds admin or teacher token in cookie/localStorage, unlock automatically without secondary password prompt
    if (typeof document !== 'undefined') {
      if (document.cookie.includes('dn_admin_token') || document.cookie.includes('dn_token') || localStorage.getItem('dn_admin_token') || localStorage.getItem('dn_token')) {
        isUnlocked.value = true;
        if (!adminToken.value) {
          adminToken.value = localStorage.getItem('dn_admin_token') || localStorage.getItem('dn_token') || '';
        }
      }
    }

    const res = await fetch('/api/config');
    if (res.ok) {
      const data = await res.json();
      config.value = data.config;
      if (config.value.fontSize) {
        notifyFontSizeChange(config.value.fontSize);
      }
      if (data.isFirstRun) {
        isFirstRun.value = true;
      }
      lanIPs.value = data.lanIPs || ['127.0.0.1'];
      frpPublicUrl.value = data.frpPublicUrl || '';
      if (data.frpStatus) {
        frpStatus.value = data.frpStatus;
      }
      if (typeof data.caInstalled === 'boolean') {
        caInstalled.value = data.caInstalled;
      }
      initSoundMode();
    }
  } catch (e) {
    console.warn('获取配置失败', e);
  }
}

function openZoom(item: AddressItem) {
  zoomedAddress.value = item;
  nextTick(() => {
    if (zoomCanvasRef.value) {
      QRCode.toCanvas(zoomCanvasRef.value, item.url, {
        width: 224,
        margin: 1,
        color: { dark: '#0f172a', light: '#ffffff' }
      }).catch(console.error);
    }
  });
}

async function unlockAdmin() {
  if (!adminPasswordInput.value) {
    adminError.value = '请输入系统统一密码';
    return;
  }

  verifyingAdmin.value = true;
  adminError.value = '';

  try {
    const res = await fetch('/api/config/verify-admin', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: adminPasswordInput.value })
    });

    if (res.ok) {
      const data = await res.json();
      adminToken.value = data.adminToken || '';
      isUnlocked.value = true;
      adminPasswordInput.value = '';
    } else {
      const msg = await res.text();
      adminError.value = msg || '密码错误，请重新输入';
    }
  } catch (e) {
    adminError.value = '网络异常，验证失败';
  } finally {
    verifyingAdmin.value = false;
  }
}

function relockAdmin() {
  isUnlocked.value = false;
  adminToken.value = '';
  newPassword.value = '';
  adminPasswordInput.value = '';
  adminError.value = '';
  if (typeof document !== 'undefined') {
    document.cookie = 'dn_admin_token=; path=/; max-age=0; expires=Thu, 01 Jan 1970 00:00:00 GMT';
  }
  fetch('/api/config/lock-admin', { method: 'POST', keepalive: true }).catch(() => {});
}

async function saveBasicConfig() {
  saving.value = true;
  try {
    const res = await fetch('/api/config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        className: config.value.className,
        soundEnabled: config.value.soundEnabled,
        customSound: config.value.customSound,
        fontSize: config.value.fontSize || '16px',
        autoStart: config.value.autoStart
      })
    });
    if (res.ok) {
      notifyFontSizeChange(config.value.fontSize || '16px');
      alert('基础设置已成功保存！');
    } else {
      alert('保存失败，请检查服务状态');
    }
  } catch (e) {
    alert('保存失败：网络连接异常');
  } finally {
    saving.value = false;
  }
}

async function saveSecurityConfig() {
  saving.value = true;
  try {
    const payload: any = {
      frp: config.value.frp,
      lanAuthRequired: config.value.lanAuthRequired
    };
    if (newPassword.value.trim()) {
      payload.newPassword = newPassword.value.trim();
      payload.newPublicPassword = newPassword.value.trim();
      payload.newAdminPassword = newPassword.value.trim();
    }

    const headers: Record<string, string> = { 'Content-Type': 'application/json' };
    if (adminToken.value) {
      headers['Authorization'] = `Bearer ${adminToken.value}`;
    }

    const res = await fetch('/api/config', {
      method: 'POST',
      headers,
      body: JSON.stringify(payload)
    });

    if (res.ok) {
      const data = await res.json().catch(() => ({}));
      if (data.adminToken) {
        adminToken.value = data.adminToken;
      }
      alert('安全与公网配置已成功保存并热重载生效！');
      newPassword.value = '';
      await fetchStatusAndConfig();
    } else {
      const msg = await res.text();
      alert(`保存失败：${msg || '权限验证不通过'}`);
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

async function closeWindow() {
  relockAdmin();
  try {
    await fetch('/api/window/settings-hide', { method: 'POST', keepalive: true });
  } catch {}
  if (typeof window !== 'undefined' && (window as any).close) {
    (window as any).close();
  }
}

// Sound handlers
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

const openingDir = ref(false);

async function openArchiveDir() {
  openingDir.value = true;
  try {
    const res = await fetch('/api/archive/open', { method: 'POST' });
    if (!res.ok) {
      const errText = await res.text();
      throw new Error(errText || `打开失败 (${res.status})`);
    }
    const data = await res.json().catch(() => ({}));
    if (data.documentsDir) {
      config.value.documentsDir = data.documentsDir;
    }
    alert('已呼出系统资源管理器打开通知存档目录');
  } catch (err: any) {
    alert(`打开存档目录失败：${err.message || '系统错误'}`);
  } finally {
    openingDir.value = false;
  }
}

function copyArchiveDir() {
  const dir = config.value.documentsDir || `我的文档\\DeskNotify\\${config.value.className || '班级通知'}`;
  navigator.clipboard.writeText(dir).then(() => {
    alert(`通知存档目录路径已复制到剪贴板：\n${dir}`);
  }).catch(() => {
    alert(`请复制路径：${dir}`);
  });
}

const openingLogsDir = ref(false);

async function openLogsDir() {
  openingLogsDir.value = true;
  try {
    const res = await fetch('/api/logs/open', { method: 'POST' });
    if (!res.ok) {
      const errText = await res.text();
      throw new Error(errText || `打开失败 (${res.status})`);
    }
    const data = await res.json().catch(() => ({}));
    if (data.logsDir) {
      config.value.logsDir = data.logsDir;
    }
    alert('已呼出系统资源管理器打开运行日志目录');
  } catch (err: any) {
    alert(`打开日志目录失败：${err.message || '系统错误'}`);
  } finally {
    openingLogsDir.value = false;
  }
}

function copyLogsDir() {
  const dir = config.value.logsDir || '%APPDATA%\\DeskNotify\\logs';
  navigator.clipboard.writeText(dir).then(() => {
    alert(`运行日志目录路径已复制到剪贴板：\n${dir}`);
  }).catch(() => {
    alert(`请复制路径：${dir}`);
  });
}

const caInstalled = ref(false);

const creatingShortcut = ref(false);

async function createDesktopShortcut() {
  creatingShortcut.value = true;
  try {
    const res = await fetch('/api/window/desktop-shortcut', { method: 'POST' });
    const data = await res.json().catch(() => ({}));
    if (res.ok) {
      alert(data.message || '已成功在电脑桌面创建【传纸条】快捷方式！');
    } else {
      alert(`创建桌面快捷方式失败：${data.error || '系统权限不足'}`);
    }
  } catch (err: any) {
    alert(`创建桌面快捷方式失败：${err.message || '系统错误'}`);
  } finally {
    creatingShortcut.value = false;
  }
}

const pinningTaskbar = ref(false);

async function pinToTaskbar() {
  pinningTaskbar.value = true;
  try {
    const res = await fetch('/api/window/pin-taskbar', { method: 'POST' });
    const data = await res.json().catch(() => ({}));
    if (!res.ok && !data.message) {
      throw new Error(`状态码 ${res.status}`);
    }
    alert(data.message || '已处理固定到底部任务栏请求。');
  } catch (err: any) {
    alert(`固定到任务栏提示：${err.message || '网络连接异常'}`);
  } finally {
    pinningTaskbar.value = false;
  }
}
</script>

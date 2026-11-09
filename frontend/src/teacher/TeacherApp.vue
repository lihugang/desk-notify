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

          <button
            @click="openSettingsModal"
            class="px-2.5 py-1 rounded-lg bg-slate-100 hover:bg-slate-200 text-slate-700 transition flex items-center gap-1.5 font-medium shadow-2xs active:scale-95"
            title="班级与系统配置"
          >
            <Settings class="w-3.5 h-3.5 text-slate-600" />
            <span>设置</span>
          </button>
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
      <div
        v-else
        class="bg-white rounded-2xl shadow-sm border overflow-hidden relative transition-all duration-150"
        :class="isDragging ? 'border-blue-500 ring-2 ring-blue-400/50 bg-blue-50/10' : 'border-slate-200'"
        @dragenter.prevent="handleDragEnter"
        @dragover.prevent="handleDragOver"
        @dragleave.prevent="handleDragLeave"
        @drop.prevent="handleDrop"
      >
        <!-- Drag & Drop Visual Overlay -->
        <div
          v-if="isDragging"
          class="absolute inset-0 z-20 bg-blue-50/95 backdrop-blur-xs border-2 border-dashed border-blue-500 rounded-2xl flex flex-col items-center justify-center p-6 text-center pointer-events-none transition-all duration-200 shadow-inner"
        >
          <div class="w-14 h-14 rounded-2xl bg-blue-100 text-blue-600 flex items-center justify-center mb-3 shadow-xs">
            <UploadCloud class="w-7 h-7 animate-bounce" />
          </div>
          <h3 class="text-base font-bold text-blue-900">松开鼠标即可插入文件</h3>
          <p class="text-xs text-blue-600 mt-1 max-w-xs leading-normal">
            图片将自动归类到图片预览 · 课件及文档将自动加入附件
          </p>
        </div>
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
            placeholder="请输入要投递给班级大屏的通知正文...（支持直接拖入文件/图片，或按 Ctrl+V 粘贴截图与文件）"
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

    <!-- Settings Modal -->
    <div
      v-if="showSettingsModal"
      class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 backdrop-blur-xs p-4 sm:p-6 overflow-y-auto"
      @click.self="closeSettingsModal"
    >
      <div class="max-w-xl w-full bg-white rounded-2xl shadow-xl border border-slate-200 overflow-hidden flex flex-col my-auto max-h-[90vh]">
        <!-- Top Bar -->
        <div class="px-5 py-3.5 bg-slate-50 border-b border-slate-200 flex items-center justify-between select-none">
          <div class="flex items-center gap-2">
            <div class="w-8 h-8 rounded-lg bg-blue-100 text-blue-600 flex items-center justify-center">
              <Settings class="w-4 h-4" />
            </div>
            <div>
              <h2 class="text-sm font-bold text-slate-800 leading-tight">系统与大屏配置</h2>
              <p class="text-[11px] text-slate-500">教师端在线调整配置（无需二次确认密码）</p>
            </div>
          </div>
          <button
            @click="closeSettingsModal"
            class="text-slate-400 hover:text-slate-600 p-1.5 rounded-lg hover:bg-slate-200/80 transition"
          >
            <X class="w-4 h-4" />
          </button>
        </div>

        <!-- Tabs -->
        <div class="flex border-b border-slate-200 bg-white">
          <button
            @click="settingsActiveTab = 'basic'"
            :class="settingsActiveTab === 'basic' ? 'border-b-2 border-blue-600 text-blue-600 font-bold bg-blue-50/40' : 'text-slate-500 hover:text-slate-700 hover:bg-slate-50'"
            class="flex-1 py-2.5 text-xs font-semibold text-center transition flex items-center justify-center gap-1.5"
          >
            <Sliders class="w-3.5 h-3.5" />
            <span>基础设置</span>
          </button>
          <button
            @click="settingsActiveTab = 'security'"
            :class="settingsActiveTab === 'security' ? 'border-b-2 border-blue-600 text-blue-600 font-bold bg-blue-50/40' : 'text-slate-500 hover:text-slate-700 hover:bg-slate-50'"
            class="flex-1 py-2.5 text-xs font-semibold text-center transition flex items-center justify-center gap-1.5"
          >
            <Shield class="w-3.5 h-3.5" />
            <span>安全与公网</span>
          </button>
          <button
            @click="settingsActiveTab = 'qrcode'"
            :class="settingsActiveTab === 'qrcode' ? 'border-b-2 border-blue-600 text-blue-600 font-bold bg-blue-50/40' : 'text-slate-500 hover:text-slate-700 hover:bg-slate-50'"
            class="flex-1 py-2.5 text-xs font-semibold text-center transition flex items-center justify-center gap-1.5"
          >
            <QrCode class="w-3.5 h-3.5" />
            <span>连接与二维码</span>
          </button>
        </div>

        <!-- Modal Body (Scrollable) -->
        <div class="p-5 overflow-y-auto space-y-5 text-xs text-slate-700">
          <!-- Tab 1: 基础设置 -->
          <div v-if="settingsActiveTab === 'basic'" class="space-y-4">
            <!-- 班级名称 -->
            <div class="space-y-1.5">
              <label class="block font-semibold text-slate-800">班级名称 / 标识</label>
              <input
                v-model="settingsConfig.className"
                type="text"
                class="w-full px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-xs focus:ring-2 focus:ring-blue-500 focus:bg-white focus:outline-none"
                placeholder="例如：2907 或 高一(3)班"
              />
              <p class="text-[11px] text-slate-400">大屏卡片顶栏与教师端显示的主要班级标识</p>
            </div>

            <!-- 提示音开关与预设选择 -->
            <div class="space-y-2 pt-2 border-t border-slate-100">
              <div class="flex items-center justify-between">
                <div>
                  <label class="block font-semibold text-slate-800">新通知到信提示音</label>
                  <p class="text-[11px] text-slate-400">大屏接收到新通知卡片时自动播放提示音</p>
                </div>
                <input
                  v-model="settingsConfig.soundEnabled"
                  type="checkbox"
                  class="w-4 h-4 text-blue-600 rounded border-slate-300 focus:ring-blue-500"
                />
              </div>

              <!-- 提示音音效选择 -->
              <div v-if="settingsConfig.soundEnabled" class="pt-1 space-y-2">
                <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
                  <div
                    v-for="sound in soundPresets"
                    :key="sound.id"
                    @click="settingsConfig.customSound = sound.id; playPreviewSound(sound.id)"
                    :class="(settingsConfig.customSound || '') === sound.id ? 'border-blue-500 bg-blue-50/50 ring-1 ring-blue-500 text-blue-900 font-semibold' : 'border-slate-200 hover:bg-slate-50 text-slate-700'"
                    class="p-2.5 rounded-xl border cursor-pointer transition flex items-center justify-between"
                  >
                    <div>
                      <div class="font-medium text-xs">{{ sound.label }}</div>
                      <div class="text-[10px] text-slate-400">{{ sound.desc }}</div>
                    </div>
                    <Volume2 class="w-3.5 h-3.5 text-slate-400" :class="(settingsConfig.customSound || '') === sound.id ? 'text-blue-600' : ''" />
                  </div>
                </div>

                <div class="flex items-center justify-between pt-1 text-[11px] text-slate-500">
                  <span>点击音效可直接在线试听</span>
                  <button
                    v-if="isPlayingSound"
                    type="button"
                    @click="stopPreviewSound"
                    class="text-red-500 hover:text-red-700 font-medium flex items-center gap-1"
                  >
                    <Square class="w-3 h-3" />
                    <span>停止播放</span>
                  </button>
                </div>
              </div>
            </div>

            <!-- 大屏显示字号 -->
            <div class="space-y-2 pt-2 border-t border-slate-100">
              <label class="block font-semibold text-slate-800">大屏卡片正文字号</label>
              <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
                <button
                  type="button"
                  v-for="item in fontSizePresets"
                  :key="item.value"
                  @click="settingsConfig.fontSize = item.value"
                  :class="(settingsConfig.fontSize || '16px') === item.value ? 'border-blue-500 bg-blue-50/60 ring-2 ring-blue-500/20 text-blue-700 font-bold' : 'border-slate-200 hover:bg-slate-50 text-slate-700'"
                  class="p-2 rounded-xl border text-center transition"
                >
                  <div class="text-xs">{{ item.label }}</div>
                  <div class="text-[10px] text-slate-400 mt-0.5">{{ item.desc }}</div>
                </button>
              </div>
              <p class="text-[11px] text-slate-400">支持 16px 标准到 26px 超大字号，保存后大屏端即时热更新</p>
            </div>

            <!-- 开机自启 -->
            <div class="flex items-center justify-between pt-2 border-t border-slate-100">
              <div>
                <label class="block font-semibold text-slate-800">开机随 Windows 自启动</label>
                <p class="text-[11px] text-slate-400">大屏电脑开机进入系统后自动在后台托盘就绪</p>
              </div>
              <input
                v-model="settingsConfig.autoStart"
                type="checkbox"
                class="w-4 h-4 text-blue-600 rounded border-slate-300 focus:ring-blue-500"
              />
            </div>
          </div>

          <!-- Tab 2: 安全与公网 -->
          <div v-if="settingsActiveTab === 'security'" class="space-y-4">
            <!-- 修改统一密码 (无需旧密码二次确认) -->
            <div class="space-y-1.5 bg-slate-50 p-3.5 rounded-xl border border-slate-200">
              <div class="flex items-center justify-between">
                <label class="block font-semibold text-slate-800">系统统一密码 (投递与管理口令)</label>
                <span class="text-[10px] text-blue-600 bg-blue-50 px-2 py-0.5 rounded-full font-medium">合一认证</span>
              </div>
              <input
                v-model="settingsNewPassword"
                type="password"
                placeholder="留空表示保持原有密码不变"
                class="w-full px-3 py-2 bg-white border border-slate-200 rounded-lg text-xs focus:ring-2 focus:ring-blue-500 focus:outline-none"
              />
              <p class="text-[11px] text-slate-400 leading-normal">
                教师端投递与配置管理使用同一套口令。如需修改，直接在此输入新密码并保存即可生效，无需二次输入确认。
              </p>
            </div>

            <!-- 局域网免密控制 -->
            <div class="flex items-center justify-between p-3.5 bg-slate-50 rounded-xl border border-slate-200">
              <div>
                <label class="block font-semibold text-slate-800">局域网内投递免密</label>
                <p class="text-[11px] text-slate-400 mt-0.5">默认开启。关闭后同一校园局域网内的设备访问也需验证密码</p>
              </div>
              <input
                :checked="!settingsConfig.lanAuthRequired"
                @change="settingsConfig.lanAuthRequired = !($event.target as HTMLInputElement).checked"
                type="checkbox"
                class="w-4 h-4 text-blue-600 rounded border-slate-300 focus:ring-blue-500"
              />
            </div>

            <!-- FRP 公网穿透配置 (支持 FreeFRP 与 OpenFrp) -->
            <div class="space-y-3 bg-slate-50 p-4 rounded-xl border border-slate-200">
              <div class="flex items-center justify-between">
                <div>
                  <div class="flex items-center gap-2 flex-wrap">
                    <span class="text-xs font-semibold text-slate-800">暴露服务到公网 (FRP / OpenFrp)</span>
                    <button
                      v-if="settingsConfig.frp && settingsConfig.frp.enabled"
                      type="button"
                      @click="showFrpImportModal = true"
                      class="px-2 py-0.5 bg-blue-50 hover:bg-blue-100 text-blue-600 rounded text-[11px] font-medium border border-blue-200/60 transition flex items-center gap-1 cursor-pointer"
                    >
                      <Download class="w-3 h-3" />
                      <span>导入配置 (TOML/INI)</span>
                    </button>
                    <button
                      v-if="settingsConfig.frp && settingsConfig.frp.enabled"
                      type="button"
                      @click="resetToFreeFRP"
                      class="px-2 py-0.5 bg-slate-100 hover:bg-slate-200 text-slate-600 rounded text-[11px] font-medium border border-slate-200/60 transition flex items-center gap-1 cursor-pointer"
                      title="重置为默认 FreeFRP 公共节点配置"
                    >
                      <RotateCcw class="w-3 h-3" />
                      <span>恢复 FreeFRP 默认</span>
                    </button>
                  </div>
                  <p class="text-[11px] text-slate-400 mt-0.5">支持内置 FreeFRP 或导入 OpenFrp 节点，开启后校外教师可在任意网络扫码投递</p>
                </div>
                <input
                  v-model="settingsConfig.frp.enabled"
                  type="checkbox"
                  class="w-4 h-4 rounded text-blue-600 cursor-pointer"
                />
              </div>

              <div v-if="settingsConfig.frp && settingsConfig.frp.enabled" class="space-y-3 bg-white p-3.5 rounded-xl border border-slate-200/80 mt-2">
                <div class="grid grid-cols-1 sm:grid-cols-3 gap-2">
                  <div class="sm:col-span-2">
                    <label class="block text-[11px] font-medium text-slate-500 mb-1">FRP 服务器地址 (Server Addr)</label>
                    <input
                      v-model="settingsConfig.frp.serverAddr"
                      type="text"
                      placeholder="如 cn-hk-bgp-4.ofalias.net 或 frp.freefrp.net"
                      class="w-full px-2.5 py-1.5 bg-slate-50 border border-slate-200 rounded text-xs focus:ring-1 focus:ring-blue-500 focus:bg-white"
                    />
                  </div>
                  <div>
                    <label class="block text-[11px] font-medium text-slate-500 mb-1">服务器端口 (Port)</label>
                    <input
                      v-model.number="settingsConfig.frp.serverPort"
                      type="number"
                      placeholder="7000 或 8120"
                      class="w-full px-2.5 py-1.5 bg-slate-50 border border-slate-200 rounded text-xs focus:ring-1 focus:ring-blue-500 focus:bg-white"
                    />
                  </div>
                </div>

                <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
                  <div>
                    <label class="block text-[11px] font-medium text-slate-500 mb-1">用户标识 User (OpenFrp 专属，普通 FRP 留空)</label>
                    <input
                      v-model="settingsConfig.frp.user"
                      type="text"
                      placeholder="OpenFrp 用户标识 (user)"
                      class="w-full px-2.5 py-1.5 bg-slate-50 border border-slate-200 rounded text-xs focus:ring-1 focus:ring-blue-500 focus:bg-white font-mono"
                    />
                  </div>
                  <div>
                    <label class="block text-[11px] font-medium text-slate-500 mb-1">通信密钥 Token</label>
                    <input
                      v-model="settingsConfig.frp.token"
                      type="password"
                      placeholder="认证 Token"
                      class="w-full px-2.5 py-1.5 bg-slate-50 border border-slate-200 rounded text-xs focus:ring-1 focus:ring-blue-500 focus:bg-white"
                    />
                  </div>
                </div>

                <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
                  <div>
                    <label class="block text-[11px] font-medium text-slate-500 mb-1">隧道名称 (Proxy Name)</label>
                    <input
                      v-model="settingsConfig.frp.proxyName"
                      type="text"
                      placeholder="如 test (留空自动生成)"
                      class="w-full px-2.5 py-1.5 bg-slate-50 border border-slate-200 rounded text-xs focus:ring-1 focus:ring-blue-500 focus:bg-white font-mono"
                    />
                  </div>
                  <div>
                    <label class="block text-[11px] font-medium text-slate-500 mb-1">远程绑定端口 (Remote Port)</label>
                    <input
                      v-model.number="settingsConfig.frp.remotePort"
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
                      v-model="settingsConfig.frp.tlsEnable"
                      type="checkbox"
                      class="w-3.5 h-3.5 rounded text-blue-600 cursor-pointer"
                    />
                  </div>

                  <div v-if="settingsConfig.frp.tlsEnable" class="flex items-center justify-between pt-1 border-t border-slate-200/40">
                    <div>
                      <span class="text-[11px] font-medium text-slate-700">禁用自定义首字节 (disableCustomTLSFirstByte)</span>
                      <p class="text-[10px] text-slate-400">OpenFrp 节点建议关闭 (false)；标准原版节点若连不上可尝试开启 (true)</p>
                    </div>
                    <input
                      v-model="settingsConfig.frp.disableCustomTLSFirstByte"
                      type="checkbox"
                      class="w-3.5 h-3.5 rounded text-blue-600 cursor-pointer"
                    />
                  </div>
                </div>

                <div class="text-[11px] text-slate-500 bg-slate-50 p-2 rounded border border-slate-200/60 flex items-center justify-between flex-wrap gap-1">
                  <div>
                    <span>外网访问地址：</span>
                    <span class="font-mono text-blue-600 font-semibold">{{ settingsFrpUrl || ('http://' + (settingsConfig.frp.serverAddr || 'frp.freefrp.net') + ':' + settingsConfig.frp.remotePort) }}</span>
                  </div>
                  <span v-if="settingsFrpStatus" :class="settingsFrpStatus === 'connected' ? 'text-emerald-600 font-medium' : 'text-slate-400'">
                    {{ settingsFrpStatus === 'connected' ? '● 已连接' : settingsFrpStatus === 'connecting' ? '◌ 连接中...' : '○ 未连接' }}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <!-- Tab 3: 投递二维码 -->
          <div v-if="settingsActiveTab === 'qrcode'" class="space-y-4">
            <div class="text-center">
              <h3 class="font-bold text-slate-800 text-sm">投递连接地址与二维码</h3>
              <p class="text-[11px] text-slate-400 mt-0.5">手机可直接扫描二维码打开教师端投递页面</p>
            </div>

            <div class="space-y-3">
              <!-- LAN Address -->
              <div v-for="ip in settingsLanIPs" :key="ip" class="p-3 bg-slate-50 border border-slate-200 rounded-xl flex items-center gap-3">
                <canvas :ref="(el) => renderSettingsQr(`http://${ip}:${settingsConfig.port || 8920}`, el)" class="w-20 h-20 bg-white p-1 rounded-lg border border-slate-200 flex-shrink-0"></canvas>
                <div class="flex-1 min-w-0">
                  <div class="flex items-center gap-1.5">
                    <span class="px-1.5 py-0.5 rounded bg-blue-100 text-blue-700 font-bold text-[10px]">局域网</span>
                    <span class="font-mono text-xs font-semibold text-slate-800 truncate">http://{{ ip }}:{{ settingsConfig.port || 8920 }}</span>
                  </div>
                  <p class="text-[10px] text-slate-400 mt-1">教室或校园 WiFi 直连，无延迟极速响应</p>
                  <button
                    type="button"
                    @click="copySettingsText(`http://${ip}:${settingsConfig.port || 8920}`)"
                    class="mt-2 text-[11px] text-blue-600 hover:text-blue-800 flex items-center gap-1 font-medium"
                  >
                    <Copy class="w-3 h-3" />
                    <span>复制投递链接</span>
                  </button>
                </div>
              </div>

              <!-- Public FRP Address -->
              <div v-if="settingsConfig.frp && settingsConfig.frp.enabled" class="p-3 bg-emerald-50/50 border border-emerald-200 rounded-xl flex items-center gap-3">
                <canvas :ref="(el) => renderSettingsQr(settingsFrpUrl || `http://${settingsConfig.frp.serverAddr}:${settingsConfig.frp.remotePort}`, el)" class="w-20 h-20 bg-white p-1 rounded-lg border border-emerald-200 flex-shrink-0"></canvas>
                <div class="flex-1 min-w-0">
                  <div class="flex items-center gap-1.5">
                    <span class="px-1.5 py-0.5 rounded bg-emerald-100 text-emerald-800 font-bold text-[10px]">公网穿透</span>
                    <span class="font-mono text-xs font-semibold text-slate-800 truncate">{{ settingsFrpUrl || `http://${settingsConfig.frp.serverAddr}:${settingsConfig.frp.remotePort}` }}</span>
                  </div>
                  <p class="text-[10px] text-slate-500 mt-1">校外 4G/5G 手机网络随时随地可投递</p>
                  <button
                    type="button"
                    @click="copySettingsText(settingsFrpUrl || `http://${settingsConfig.frp.serverAddr}:${settingsConfig.frp.remotePort}`)"
                    class="mt-2 text-[11px] text-emerald-700 hover:text-emerald-900 flex items-center gap-1 font-medium"
                  >
                    <Copy class="w-3 h-3" />
                    <span>复制公网链接</span>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Modal Footer -->
        <div class="px-5 py-3 bg-slate-50 border-t border-slate-200 flex items-center justify-end gap-2.5">
          <button
            type="button"
            @click="closeSettingsModal"
            class="px-4 py-2 text-xs font-medium text-slate-600 hover:bg-slate-200/80 rounded-xl transition"
          >
            取消
          </button>
          <button
            type="button"
            @click="saveTeacherSettings"
            :disabled="settingsSaving"
            class="px-5 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-xl text-xs font-bold shadow-xs disabled:opacity-50 transition flex items-center gap-1.5"
          >
            <Check class="w-3.5 h-3.5" />
            <span>{{ settingsSaving ? '正在保存...' : '保存配置' }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- FRP Import Modal -->
    <div
      v-if="showFrpImportModal"
      class="fixed inset-0 bg-slate-900/60 backdrop-blur-xs flex items-center justify-center p-4 z-[60] animate-fade-in"
      @click.self="showFrpImportModal = false"
    >
      <div class="bg-white rounded-2xl max-w-lg w-full p-5 space-y-4 shadow-2xl border border-slate-100">
        <div class="flex items-center justify-between border-b border-slate-100 pb-3">
          <div class="flex items-center gap-2">
            <Download class="w-4 h-4 text-blue-600" />
            <h3 class="text-sm font-bold text-slate-800">一键导入 FRP / OpenFrp 配置</h3>
          </div>
          <button type="button" @click="showFrpImportModal = false" class="p-1 hover:bg-slate-100 rounded-lg text-slate-400 hover:text-slate-700 cursor-pointer">
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
            type="button"
            @click="showFrpImportModal = false"
            class="px-4 py-2 text-xs font-medium text-slate-600 hover:bg-slate-100 rounded-xl transition cursor-pointer"
          >
            取消
          </button>
          <button
            type="button"
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
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue';
import {
  Bell, Lock, Bold, Italic, Heading2, List, Quote,
  ImageIcon, Paperclip, Send, X, FileText, Pin, ZoomIn, Eye, Edit3,
  UploadCloud, Settings, Sliders, Shield, QrCode, Volume2, Square,
  Check, Copy, Download, RotateCcw
} from 'lucide-vue-next';
import { marked } from 'marked';
import QRCode from 'qrcode';
import HistoryList from '../components/HistoryList.vue';
import ImageLightbox from '../components/ImageLightbox.vue';
import { spaceText, spacePage } from '../utils/pangu';
import { parseFRPConfigText } from '../utils/frpParser';

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
const isPinned = ref(false);
const duration = ref(0);
const images = ref<UploadItem[]>([]);
const attachments = ref<UploadItem[]>([]);
const submitting = ref(false);
const historyListRef = ref<InstanceType<typeof HistoryList> | null>(null);
const textareaRef = ref<HTMLTextAreaElement | null>(null);

const activeLightboxImage = ref<string | null>(null);
const isPreview = ref(false);
const isDragging = ref(false);
const dragCounter = ref(0);

const renderedMarkdown = computed(() => {
  if (!textContent.value.trim()) {
    return '<p class="text-slate-400 italic">（暂无正文内容）</p>';
  }
  const html = marked.parse(textContent.value, { breaks: true, gfm: true }) as string;
  nextTick(() => spacePage());
  return html;
});

watch(isPreview, async (val) => {
  if (val) {
    await nextTick();
    spacePage();
  }
});

function previewImage(url?: string) {
  if (url) {
    activeLightboxImage.value = url;
  }
}

function preventWindowDrop(e: DragEvent) {
  e.preventDefault();
}

onMounted(() => {
  checkAuthAndStatus();
  window.addEventListener('dragover', preventWindowDrop);
  window.addEventListener('drop', preventWindowDrop);
});

onUnmounted(() => {
  stopPreviewSound();
  images.value.forEach(img => {
    if (img.preview) URL.revokeObjectURL(img.preview);
  });
  window.removeEventListener('dragover', preventWindowDrop);
  window.removeEventListener('drop', preventWindowDrop);
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
      const data = await res.json().catch(() => ({}));
      if (data.token) {
        localStorage.setItem('dn_token', data.token);
      }
      if (data.adminToken) {
        localStorage.setItem('dn_admin_token', data.adminToken);
      }
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

// Settings Modal State & Handlers
const showSettingsModal = ref(false);
const settingsActiveTab = ref<'basic' | 'security' | 'qrcode'>('basic');
const settingsConfig = ref<any>({
  className: '',
  port: 8920,
  soundEnabled: true,
  customSound: '',
  fontSize: '16px',
  autoStart: true,
  lanAuthRequired: false,
  frp: {
    enabled: false,
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
const settingsLanIPs = ref<string[]>([]);
const settingsFrpUrl = ref('');
const settingsFrpStatus = ref('');
const settingsSaving = ref(false);
const settingsNewPassword = ref('');
const isPlayingSound = ref(false);
let previewAudio: HTMLAudioElement | null = null;

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

  if (!settingsConfig.value.frp) {
    settingsConfig.value.frp = {
      enabled: true,
      serverAddr: '',
      serverPort: 7000,
      token: '',
      remotePort: 0,
      user: '',
      proxyName: '',
      protocol: 'tcp',
      tlsEnable: false,
      disableCustomTLSFirstByte: false
    };
  }

  if (parsed.serverAddr) settingsConfig.value.frp.serverAddr = parsed.serverAddr;
  if (parsed.serverPort) settingsConfig.value.frp.serverPort = parsed.serverPort;
  if (parsed.user !== undefined) settingsConfig.value.frp.user = parsed.user;
  if (parsed.token) settingsConfig.value.frp.token = parsed.token;
  if (parsed.remotePort) settingsConfig.value.frp.remotePort = parsed.remotePort;
  if (parsed.proxyName !== undefined) settingsConfig.value.frp.proxyName = parsed.proxyName;
  if (parsed.protocol) settingsConfig.value.frp.protocol = parsed.protocol;
  if (parsed.tlsEnable !== undefined) settingsConfig.value.frp.tlsEnable = parsed.tlsEnable;
  if (parsed.disableCustomTLSFirstByte !== undefined) {
    settingsConfig.value.frp.disableCustomTLSFirstByte = parsed.disableCustomTLSFirstByte;
  }
  settingsConfig.value.frp.enabled = true;

  showFrpImportModal.value = false;
  frpImportText.value = '';
  frpImportError.value = '';
  alert('配置导入并解析成功！已自动填充表单各项参数，请点击“保存配置”以生效。');
}

function resetToFreeFRP() {
  if (!settingsConfig.value.frp) {
    settingsConfig.value.frp = {
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
    };
  }
  settingsConfig.value.frp.serverAddr = 'frp.freefrp.net';
  settingsConfig.value.frp.serverPort = 7000;
  settingsConfig.value.frp.token = 'freefrp.net';
  settingsConfig.value.frp.user = '';
  settingsConfig.value.frp.proxyName = '';
  settingsConfig.value.frp.protocol = 'tcp';
  settingsConfig.value.frp.tlsEnable = false;
  settingsConfig.value.frp.disableCustomTLSFirstByte = false;
  if (!settingsConfig.value.frp.remotePort) {
    settingsConfig.value.frp.remotePort = 10001 + Math.floor(Math.random() * 39999);
  }
  alert('已重置为默认 FreeFRP 节点配置，请点击“保存配置”以应用。');
}

const soundPresets = [
  { id: 'glass.mp3', label: '清脆风铃', desc: '灵动通透' },
  { id: 'minimal.mp3', label: '极简提示', desc: '干脆利落' },
  { id: 'soft.mp3', label: '柔和气泡', desc: '温润悦耳' },
  { id: 'polite.mp3', label: '优雅轻响', desc: '经典现代' },
  { id: 'chime.mp3', label: '空灵旋律', desc: '舒缓悠扬' },
  { id: '', label: '系统默认', desc: '经典提示音' },
];

const fontSizePresets = [
  { value: '16px', label: '标准 16px', desc: '默认推荐' },
  { value: '18px', label: '较大 18px', desc: '中等教室' },
  { value: '22px', label: '特大 22px', desc: '宽敞大班' },
  { value: '26px', label: '超大 26px', desc: '阶梯教室' },
];

function closeSettingsModal() {
  stopPreviewSound();
  showSettingsModal.value = false;
}

function playPreviewSound(soundId: string) {
  stopPreviewSound();
  const url = soundId ? `/sounds/${soundId}` : `/alert.mp3`;
  try {
    previewAudio = new Audio(url);
    isPlayingSound.value = true;
    previewAudio.onended = () => {
      isPlayingSound.value = false;
      previewAudio = null;
    };
    previewAudio.onerror = () => {
      isPlayingSound.value = false;
      previewAudio = null;
    };
    previewAudio.play().catch(() => {
      isPlayingSound.value = false;
      previewAudio = null;
    });
  } catch (e) {
    isPlayingSound.value = false;
    previewAudio = null;
  }
}

function stopPreviewSound() {
  if (previewAudio) {
    previewAudio.pause();
    previewAudio.currentTime = 0;
    previewAudio = null;
  }
  isPlayingSound.value = false;
}

function copySettingsText(text: string) {
  navigator.clipboard.writeText(text).then(() => {
    alert('已复制链接到剪贴板：' + text);
  }).catch(() => {
    alert('请手动复制链接：' + text);
  });
}

function renderSettingsQr(url: string, el: any) {
  if (el && url) {
    QRCode.toCanvas(el as HTMLCanvasElement, url, {
      width: 80,
      margin: 1,
      color: { dark: '#0f172a', light: '#ffffff' }
    }).catch(() => {});
  }
}

async function openSettingsModal() {
  try {
    const res = await fetch('/api/config');
    if (res.ok) {
      const data = await res.json();
      settingsConfig.value = JSON.parse(JSON.stringify(data.config || {}));
      settingsConfig.value.frp = {
        enabled: false,
        serverAddr: 'frp.freefrp.net',
        serverPort: 7000,
        token: 'freefrp.net',
        remotePort: 28932,
        user: '',
        proxyName: '',
        protocol: 'tcp',
        tlsEnable: false,
        disableCustomTLSFirstByte: false,
        ...(settingsConfig.value.frp || {})
      };
      settingsLanIPs.value = data.lanIPs || ['127.0.0.1'];
      settingsFrpUrl.value = data.frpPublicUrl || '';
      settingsFrpStatus.value = data.frpStatus || '';
      settingsNewPassword.value = '';
      showSettingsModal.value = true;
    } else {
      alert('获取系统配置失败');
    }
  } catch (e) {
    alert('网络连接错误，无法加载配置');
  }
}

async function saveTeacherSettings() {
  settingsSaving.value = true;
  stopPreviewSound();
  try {
    const payload: any = {
      className: settingsConfig.value.className,
      soundEnabled: settingsConfig.value.soundEnabled,
      customSound: settingsConfig.value.customSound,
      fontSize: settingsConfig.value.fontSize || '16px',
      autoStart: settingsConfig.value.autoStart,
      lanAuthRequired: settingsConfig.value.lanAuthRequired,
      frp: settingsConfig.value.frp
    };

    if (settingsNewPassword.value.trim()) {
      payload.newPassword = settingsNewPassword.value.trim();
    }

    const headers: Record<string, string> = { 'Content-Type': 'application/json' };
    const savedAdminToken = localStorage.getItem('dn_admin_token') || localStorage.getItem('dn_token');
    if (savedAdminToken) {
      headers['Authorization'] = `Bearer ${savedAdminToken}`;
    }

    const res = await fetch('/api/config', {
      method: 'POST',
      headers,
      body: JSON.stringify(payload)
    });

    if (res.ok) {
      const data = await res.json().catch(() => ({}));
      if (data.adminToken) {
        localStorage.setItem('dn_admin_token', data.adminToken);
      }
      if (data.token) {
        localStorage.setItem('dn_token', data.token);
      }
      if (data.frpStatus) {
        settingsFrpStatus.value = data.frpStatus;
      }
      if (data.frpPublicUrl) {
        settingsFrpUrl.value = data.frpPublicUrl;
      }
      classroomName.value = settingsConfig.value.className || classroomName.value;
      showSettingsModal.value = false;
      alert('配置已成功保存并实时生效！');
    } else {
      const msg = await res.text();
      alert(`保存失败：${msg || '服务器错误'}`);
    }
  } catch (e) {
    alert('网络连接异常，保存失败');
  } finally {
    settingsSaving.value = false;
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

function isImageFile(file: File): boolean {
  if (file.type && file.type.startsWith('image/')) {
    return true;
  }
  const ext = file.name.substring(file.name.lastIndexOf('.')).toLowerCase();
  return ['.jpg', '.jpeg', '.png', '.gif', '.webp', '.svg', '.bmp', '.ico', '.avif'].includes(ext);
}

function processFiles(files: File[]) {
  if (!files || files.length === 0) return;

  // Auto switch back to edit mode if currently previewing
  if (isPreview.value) {
    isPreview.value = false;
  }

  for (const file of files) {
    if (isImageFile(file)) {
      const exists = images.value.some(img => img.name === file.name && img.size === file.size);
      if (!exists) {
        images.value.push({
          file,
          preview: URL.createObjectURL(file),
          name: file.name || `image_${Date.now()}.png`,
          size: file.size
        });
      }
    } else {
      const exists = attachments.value.some(att => att.name === file.name && att.size === file.size);
      if (!exists) {
        attachments.value.push({
          file,
          name: file.name,
          size: file.size
        });
      }
    }
  }
}

function handleDragEnter(e: DragEvent) {
  e.preventDefault();
  if (e.dataTransfer && e.dataTransfer.types.includes('Files')) {
    dragCounter.value++;
    isDragging.value = true;
  }
}

function handleDragOver(e: DragEvent) {
  if (e.dataTransfer && e.dataTransfer.types.includes('Files')) {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'copy';
  }
}

function handleDragLeave() {
  dragCounter.value--;
  if (dragCounter.value <= 0) {
    dragCounter.value = 0;
    isDragging.value = false;
  }
}

function handleDrop(e: DragEvent) {
  e.preventDefault();
  dragCounter.value = 0;
  isDragging.value = false;

  const files = e.dataTransfer?.files;
  if (!files || files.length === 0) return;

  processFiles(Array.from(files));
}

function handlePaste(e: ClipboardEvent) {
  const clipboardFiles = e.clipboardData?.files;
  if (clipboardFiles && clipboardFiles.length > 0) {
    e.preventDefault();
    processFiles(Array.from(clipboardFiles));
    return;
  }

  const items = e.clipboardData?.items;
  if (items) {
    for (let i = 0; i < items.length; i++) {
      if (items[i].type.indexOf('image') !== -1) {
        const file = items[i].getAsFile();
        if (file) {
          e.preventDefault();
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
}

function handleImageSelect(e: Event) {
  const target = e.target as HTMLInputElement;
  if (!target.files) return;
  processFiles(Array.from(target.files));
  target.value = '';
}

function handleFileSelect(e: Event) {
  const target = e.target as HTMLInputElement;
  if (!target.files) return;
  processFiles(Array.from(target.files));
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
    // 发送通知时自动应用盘古之白
    const formattedContent = spaceText(textContent.value);
    textContent.value = formattedContent;

    const formData = new FormData();
    formData.append('content', formattedContent);
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
      isPinned.value = false;
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

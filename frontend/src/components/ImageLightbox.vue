<template>
  <Teleport to="body">
    <Transition name="fade">
      <div
        v-if="imageUrl"
        class="fixed inset-0 z-50 bg-black/85 backdrop-blur-xs flex items-center justify-center p-4 select-none"
        @click="close"
        @keydown.esc="close"
        tabindex="0"
      >
        <!-- Top Toolbar -->
        <div class="absolute top-4 right-4 flex items-center gap-3 z-10" @click.stop>
          <a
            :href="imageUrl"
            target="_blank"
            download
            class="p-2 rounded-full bg-white/20 hover:bg-white/30 text-white transition flex items-center justify-center"
            title="下载原图"
          >
            <Download class="w-5 h-5" />
          </a>
          <a
            :href="imageUrl"
            target="_blank"
            class="p-2 rounded-full bg-white/20 hover:bg-white/30 text-white transition flex items-center justify-center"
            title="新窗口查看"
          >
            <ExternalLink class="w-5 h-5" />
          </a>
          <button
            @click="close"
            class="p-2 rounded-full bg-white/20 hover:bg-red-500/80 text-white transition flex items-center justify-center cursor-pointer"
            title="关闭 (Esc)"
          >
            <X class="w-5 h-5" />
          </button>
        </div>

        <!-- Image Container -->
        <div class="relative max-w-full max-h-full flex items-center justify-center" @click.stop>
          <img
            :src="imageUrl"
            class="max-w-[92vw] max-h-[88vh] object-contain rounded-xl shadow-2xl transition-transform duration-200"
            alt="预览大图"
          />
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue';
import { X, Download, ExternalLink } from 'lucide-vue-next';

const props = defineProps<{
  imageUrl: string | null;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
}>();

function close() {
  emit('close');
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.imageUrl) {
    close();
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown);
});
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>

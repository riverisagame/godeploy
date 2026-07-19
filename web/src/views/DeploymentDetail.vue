<template>
  <div class="h-full flex flex-col p-6 space-y-6">
    <div class="flex justify-between items-center bg-gray-800 p-4 rounded-xl shadow-lg border border-gray-700/50">
      <div>
        <h2 class="text-2xl font-bold bg-gradient-to-r from-emerald-400 to-teal-400 bg-clip-text text-transparent">
          Deployment Execution
        </h2>
        <p class="text-gray-400 text-sm mt-1">Status: <span class="font-mono text-emerald-400">Running</span></p>
      </div>
      <div>
        <button @click="$router.back()" class="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-gray-200 rounded-lg transition-colors">
          Back
        </button>
      </div>
    </div>

    <!-- Terminal Window -->
    <div class="flex-1 bg-gray-950 rounded-xl border border-gray-800 shadow-2xl overflow-hidden flex flex-col font-mono text-sm">
      <!-- Terminal Header -->
      <div class="bg-gray-900 border-b border-gray-800 px-4 py-2 flex items-center space-x-2">
        <div class="w-3 h-3 rounded-full bg-red-500/80"></div>
        <div class="w-3 h-3 rounded-full bg-yellow-500/80"></div>
        <div class="w-3 h-3 rounded-full bg-green-500/80"></div>
        <span class="ml-4 text-gray-500 text-xs font-sans">Deployment Console</span>
      </div>
      
      <!-- Logs Area -->
      <div class="flex-1 overflow-y-auto p-4 space-y-1" ref="logContainer">
        <div v-for="(log, idx) in logs" :key="idx" class="text-gray-300">
          <span class="text-gray-600 mr-4 select-none">{{ idx + 1 }}</span>
          <span v-if="log.includes('ERROR')" class="text-red-400">{{ log }}</span>
          <span v-else-if="log.includes('>>>')" class="text-cyan-400">{{ log }}</span>
          <span v-else>{{ log }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue';
import { useRoute } from 'vue-router';

const route = useRoute();
const logs = ref<string[]>([]);
const logContainer = ref<HTMLElement | null>(null);
let eventSource: EventSource | null = null;

const scrollToBottom = () => {
  nextTick(() => {
    if (logContainer.value) {
      logContainer.value.scrollTop = logContainer.value.scrollHeight;
    }
  });
};

const connectSSE = () => {
  const deployID = route.params.id;
  if (!deployID) return;

  eventSource = new EventSource(`/api/deployments/${deployID}/logs`);
  
  eventSource.onmessage = (event) => {
    if (event.data === '[EOF]') {
      if (eventSource) eventSource.close();
      logs.value.push('>>> Connection closed. Deployment finished.');
      scrollToBottom();
      return;
    }
    logs.value.push(event.data);
    scrollToBottom();
  };

  eventSource.onerror = (error) => {
    console.error('SSE Error:', error);
    logs.value.push('>>> Error connecting to log stream.');
    if (eventSource) eventSource.close();
    scrollToBottom();
  };
};

onMounted(() => {
  connectSSE();
});

onUnmounted(() => {
  if (eventSource) {
    eventSource.close();
  }
});
</script>

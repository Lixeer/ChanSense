<template>
  <div class="flex flex-row text-black h-screen overflow-hidden">
    <aside class="p-4 flex border-r-2">
      <div class="flex flex-col gap-4 w-64 h-full">
        <div class="flex gap-2">
          <label class="">Serial Port</label>
          <input class="border w-32" v-model="state.name" placeholder="e.g. COM4" />
        </div>

        <div class="flex gap-2">
          <label class="">Baud Rate</label>
          <select class="border w-32" name="" id="" v-model="state.baud">
            <option value="9600">9600</option>
            <option value="19200">19200</option>
            <option value="38400">38400</option>
            <option value="57600">57600</option>
            <option value="115200" selected>115200</option>
          </select>
        </div>

        <div class="flex justify-between gap-2">
          <button class="border w-full rounded hover:bg-gray-200 active:bg-gray-300" @click="handleUpdate">
            应用配置
          </button>
          <button class="border w-full rounded hover:bg-gray-200 active:bg-gray-300" @click="handleReconnect">
            重新连接
          </button>
        </div>

        <div class="flex flex-col justify-center">
          <span class="">连接状态</span>
          <div class="flex items-center gap-1">
            <div class="w-2 h-2 rounded-full" :class="state.isConnected ? 'bg-green-500' : 'bg-red-400'"></div>
            <span class="font-mono font-bold" :class="state.isConnected ? 'text-green-700' : 'text-gray-700'">
              {{ goLog }}
            </span>
          </div>
        </div>
      </div>
    </aside>
    <main class="flex-1 overflow-y-auto gap-4 p-4 pb-32">
      <h2>CSI 数据监控</h2>
      <!-- <div>RSSI: {{ csiData.rssi }}</div> -->
      <div>Index: {{ csiData.index }}</div>
      <div>子载波数量: {{ csiData.amplitude?.length }}</div>

      <!-- 幅度图表 -->
      <Amplitude :amplitude="csiData.amplitude || []" />

      <!-- 相位折线图 -->
      <Phase :phase="csiData.phase || []" />

      <!-- 相位差图 -->
      <PhaseDifference :phase="csiData.phase || []" />

      <!-- 相位极坐标图 -->
      <Polar :amplitude="csiData.amplitude || []" :phase="csiData.phase || []" />

      <!-- 幅度瀑布图 -->
      <AmplitudeWaterfall :amplitude="csiData.amplitude || []" />
    </main>
  </div>
</template>

<script setup lang="js">
import { ref, onMounted, onUnmounted, reactive } from "vue";
import { EventsOn, EventsOff } from "../wailsjs/runtime";
import { UpdateSerialConfig, Reconnect } from "../wailsjs/go/main/App";
import Amplitude from "./components/Amplitude.vue";
import Phase from "./components/Phase.vue";
import PhaseDifference from "./components/PhaseDifference.vue";
import Polar from "./components/Polar.vue";
import AmplitudeWaterfall from "./components/AmplitudeWaterfall.vue";

const csiData = ref({});

const state = reactive({
  name: "COM4", // 对应 app.go 中的 serialConfig.Name
  baud: 115200, // 对应 app.go 中的 serialConfig.Baud
  isConnected: false,
});
const goLog = ref("");
let unsubscribe = null;

const handleUpdate = async () => {
  // 调用 app.go 中的 UpdateSerialConfig
  await UpdateSerialConfig(state.name, Number(state.baud));
};

const handleReconnect = async () => {
  // 调用 app.go 中的 Reconnect
  await Reconnect();
};

onMounted(() => {
  // 监听连接状态事件
  EventsOn("connection-status", (status) => {
    goLog.value = `[${new Date().toLocaleTimeString()}] ${status}\n`;
    console.log("连接状态:", status);
    if (status.includes("连接成功")) {
      state.isConnected = true;
    } else {
      state.isConnected = false;
    }
  });
});

onMounted(() => {
  // 监听 Go 发送的 "csi-data" 事件
  unsubscribe = EventsOn("csi-data", (frame) => {
    // console.log("收到 CSI:", frame);
    csiData.value = frame;
    state.isConnected = true;
  });
});

onUnmounted(() => {
  // 取消监听，防止内存泄漏
  if (unsubscribe) unsubscribe();
});
</script>

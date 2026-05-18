<template>
  <div class="flex flex-row text-black h-screen overflow-hidden">
    <aside class="p-4 flex border-r-2">
      <div class="flex flex-col gap-4 w-64 h-full">
        <!-- 串口 -->
        <div class="flex gap-2">
          <label class="">Serial Port</label>
          <input class="border w-32" v-model.number="state.name" placeholder="e.g. COM4" />
        </div>

        <!-- 码率 -->
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

        <!-- 历史数据点数 -->
        <div class="flex gap-2">
          <span>历史数据点数:</span>
          <input class="border w-16" v-model.number="MAX_HISTORY" type="number" placeholder="历史数据点数" />
        </div>

        <!-- 配置 -->
        <div class="flex justify-between gap-2">
          <button class="border w-full rounded hover:bg-gray-200 active:bg-gray-300" @click="handleUpdate">
            应用配置
          </button>
          <button class="border w-full rounded hover:bg-gray-200 active:bg-gray-300" @click="handleReconnect">
            重新连接
          </button>
        </div>

        <!-- 状态响应 -->
        <div class="flex flex-col justify-center">
          <span class="">连接状态</span>
          <div class="flex items-center gap-1">
            <div class="w-2 h-2 rounded-full" :class="state.isConnected ? 'bg-green-500' : 'bg-red-400'"></div>
            <span class="font-mono font-bold" :class="state.isConnected ? 'text-green-700' : 'text-gray-700'">
              {{ goLog }}
            </span>
          </div>
        </div>

        <!-- 组件可见性控制 -->
        <div class="flex justify-between gap-2">
          <button
            class="border w-full rounded hover:bg-gray-200 active:bg-gray-300"
            :class="{ active: currentMode === 'edit' }"
            @click="currentMode = 'edit'"
          >
            编辑标记
          </button>
          <button
            class="border w-full rounded hover:bg-gray-200 active:bg-gray-300"
            :class="{ active: currentMode === 'view' }"
            @click="currentMode = 'view'"
          >
            数据源监控
          </button>
        </div>

        <!-- 组件可见性控制 -->
        <div class="flex flex-row flex-wrap gap-4" v-if="currentMode === 'view'">
          <label v-for="item in componentOptions" :key="item.id" class="checkbox-label">
            <input type="checkbox" :value="item.value" v-model="visibleComponents" />
            <span>{{ item.label }}</span>
          </label>
        </div>

        <!-- 数据标签 -->
        <div class="" v-if="currentMode === 'edit'">
          <div class="flex gap-2">
            <input
              type="text"
              class="border w-32"
              v-model="newLabel"
              placeholder="输入标签名称"
              @keyup.enter="handleSetLabel"
            />
            <button class="border w-16 rounded hover:bg-gray-200 active:bg-gray-300 text-sm" @click="handleSetLabel">
              添加标签
            </button>
            <button
              class="border w-8 rounded hover:bg-gray-200 active:bg-gray-300 text-sm"
              @click="labelStore.clearLabels"
            >
              清空
            </button>
          </div>
          <div>
            <span class="text-sm text-gray-500">暂无标签</span>
            <div class="flex flex-wrap gap-1 mt-1">
              <button
                v-for="label in labelStore.labels"
                :key="label"
                class="px-2 py-1 bg-blue-100 text-blue-800 rounded text-sm"
                :class="{ 'ring-2 ring-blue-500': label === labelStore.activeLabel }"
                @click="labelStore.setActiveLabel(label)"
              >
                {{ label }}
              </button>
            </div>
          </div>
        </div>

        <!-- 保存 -->
        <div>
          <button class="border w-full rounded hover:bg-gray-200 active:bg-gray-300" @click="saveToFile">
            即时保存{{ MAX_HISTORY }}条数据
          </button>
        </div>

        <!-- 注释 -->
        <div class="mt-auto">
          <span :class="isListening ? 'text-green-500' : 'text-red-400'">空格暂停输入</span>
        </div>
      </div>
    </aside>
    <main class="flex-1 overflow-y-auto gap-4 p-4 pb-32">
      <div class="flex flex-col gap-8" v-if="currentMode === 'view'">
        <div>
          <h2>CSI 数据监控</h2>
          <!-- <div>RSSI: {{ csiData.rssi }}</div> -->
          <div>Index: {{ csiData.index }}</div>
          <div>子载波数量: {{ csiData.amplitude?.length }}</div>
        </div>

        <!-- 幅度图表 -->
        <Amplitude :amplitude="csiData.amplitude || []" v-if="visibleComponents.includes('amplitude')" />

        <!-- 相位折线图 -->
        <Phase :phase="csiData.phase || []" v-if="visibleComponents.includes('phase')" />

        <!-- 相位差图 -->
        <PhaseDifference :phase="csiData.phase || []" v-if="visibleComponents.includes('phaseDiff')" />

        <!-- 相位极坐标图 -->
        <Polar
          :amplitude="csiData.amplitude || []"
          :phase="csiData.phase || []"
          v-if="visibleComponents.includes('polar')"
        />

        <!-- 幅度瀑布图 -->
        <AmplitudeWaterfall
          :history="MAX_HISTORY || 50"
          :amplitude="csiData.amplitude || []"
          v-if="visibleComponents.includes('waterfall')"
        />
      </div>
      <div v-else-if="currentMode === 'edit'">
        <h2>编辑标记</h2>
        <AmplitudeWaterfall :history="MAX_HISTORY || 50" :amplitude="csiData.amplitude || []" />
      </div>
      <div v-else>
        <h2>请选择一个模式</h2>
      </div>
    </main>
  </div>
</template>

<script setup lang="js">
import { ref, onMounted, onUnmounted, reactive } from "vue";
import { EventsOn, EventsOff } from "../wailsjs/runtime";
import { UpdateSerialConfig, Reconnect, AutoSaveTextToFile } from "../wailsjs/go/main/App";
import Amplitude from "./components/Amplitude.vue";
import Phase from "./components/Phase.vue";
import PhaseDifference from "./components/PhaseDifference.vue";
import Polar from "./components/Polar.vue";
import AmplitudeWaterfall from "./components/AmplitudeWaterfall.vue";
import { useMagicKeys, whenever, onKeyStroke, useDateFormat } from "@vueuse/core";
import { useLabelStore } from "./stores/Label";

const csiData = ref({});
const MAX_HISTORY = ref(200);

const state = reactive({
  name: "COM35", // 对应 app.go 中的 serialConfig.Name
  baud: 115200, // 对应 app.go 中的 serialConfig.Baud
  isConnected: false,
});
const goLog = ref("");
let unsubscribe = null;
const componentOptions = [
  { id: "amplitude", label: "幅度", value: "amplitude" },
  { id: "phase", label: "相位", value: "phase" },
  { id: "phaseDiff", label: "相位差", value: "phaseDiff" },
  { id: "polar", label: "极坐标", value: "polar" },
  { id: "waterfall", label: "幅度瀑布", value: "waterfall" },
];
const visibleComponents = ref(componentOptions.map((item) => item.value)); // 默认全部显示
const currentMode = ref("view"); // 默认展示模式

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

// 空格监听
onKeyStroke(" ", () => {
  isListening.value = !isListening.value;
});
const isListening = ref(true);
onMounted(() => {
  // 监听 Go 发送的 "csi-data" 事件
  unsubscribe = EventsOn("csi-data", (frame) => {
    // console.log("收到 CSI:", frame);
    if (isListening.value) {
      csiData.value = frame;
    } else {
      // csiData.value = {};
    }
    state.isConnected = true;
  });
});

onUnmounted(() => {
  // 取消监听，防止内存泄漏
  if (unsubscribe) unsubscribe();
});

async function saveToFile() {
  const formattedTime = useDateFormat(new Date(), "YYYYMMDD_HHmmss");
  const count = MAX_HISTORY.value;

  const filename = labelStore.activeLabel
    ? `${labelStore.activeLabel}_${formattedTime.value}.json`
    : `${formattedTime.value}.json`;

  try {
    await AutoSaveTextToFile(csiData.value.index, count, filename); // 调用 Go 方法并传递变量
    console.log("保存成功，文件名：", filename);
  } catch (e) {
    alert("保存失败：" + e);
  }
}

const labelStore = useLabelStore();
// 新标签输入
const newLabel = ref("");
function handleSetLabel() {
  const trimmedValue = newLabel.value.trim();
  if (!trimmedValue) {
    alert("标签内容不能为空！");
    return;
  }
  if (labelStore.labels.includes(trimmedValue)) {
    alert("该标签已存在！");
    return;
  }
  labelStore.addLabel(trimmedValue);

  labelStore.setActiveLabel(trimmedValue);
  newLabel.value = "";
}
</script>

<style scoped>
.active {
  background: rgb(229 231 235);
}
</style>

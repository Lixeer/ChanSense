<template>
  <div>
    <section flex justify-center gap-4 items-end bg-white p-5 text-black shadow-sm border="1 gray-100">
      <div flex flex-col gap-1>
        <label text-xs font-bold text-gray-400 uppercase tracking-wider>Serial Port</label>
        <input
          v-model="state.name"
          type="text"
          placeholder="e.g. COM4"
          border="1 gray-200"
          rounded-md
          px-3
          py-2
          w-36
          focus:ring-2
          focus:ring-blue-500
          outline-none
          transition-all
        />
      </div>

      <div flex flex-col gap-1>
        <label text-xs font-bold text-gray-400 uppercase tracking-wider>Baud Rate</label>
        <input
          v-model="state.baud"
          type="number"
          placeholder="115200"
          border="1 gray-200"
          rounded-md
          px-3
          py-2
          w-36
          focus:ring-2
          focus:ring-blue-500
          outline-none
          transition-all
        />
      </div>

      <button
        px-5
        py-2
        bg-blue-600
        text-white
        rounded-md
        font-medium
        hover:bg-blue-700
        active:scale-95
        transition-all
        @click="handleUpdate"
      >
        应用配置
      </button>

      <button
        px-5
        py-2
        border="1 blue-600"
        text-blue-600
        rounded-md
        font-medium
        hover:bg-blue-50
        active:scale-95
        transition-all
        @click="handleReconnect"
      >
        重新连接
      </button>

      <div p-4 rounded-lg border="1 dashed gray-200" flex flex-col justify-center>
        <span text-xs text-gray-400>连接状态</span>
        <div flex items-center gap-2 mt-1>
          <div w-2 h-2 rounded-full :class="state.isConnected ? 'bg-green-500' : 'bg-red-400'"></div>
          <span font-mono font-bold :class="state.isConnected ? 'text-green-700' : 'text-gray-500'">
            {{ goLog }}
          </span>
        </div>
      </div>
    </section>
    <h2>CSI 数据监控</h2>
    <div>RSSI: {{ csiData.rssi }}</div>
    <div>Index: {{ csiData.index }}</div>
    <div>子载波数量: {{ csiData.amplitude?.length }}</div>

    <!-- 幅度图表 -->
    <div class="chart-container" wfull flex flex-col gap-2 justify-center items-center>
      <h3>幅度 (Amplitude)</h3>
      <canvas ref="amplitudeChart" width="800" height="200"></canvas>
    </div>

    <!-- 相位折线图 -->
    <div class="chart-container" wfull flex flex-col gap-2 justify-center items-center>
      <h3>相位 (Phase)</h3>
      <canvas ref="phaseChart" width="800" height="200"></canvas>
    </div>

    <!-- 相位差图 -->
    <div class="chart-container" wfull flex flex-col gap-2 justify-center items-center>
      <h3>相位差 (Phase Difference) - 与上一帧对比</h3>
      <canvas ref="phaseDiffChart" width="800" height="200"></canvas>
    </div>

    <!-- 相位极坐标图 -->
    <div class="chart-container" wfull flex flex-col gap-2 justify-center items-center>
      <h3>相位极坐标 (Polar)</h3>
      <canvas ref="polarChart" width="400" height="400"></canvas>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, reactive } from "vue";
import { EventsOn, EventsOff } from "../wailsjs/runtime";
import { UpdateSerialConfig, Reconnect } from "../wailsjs/go/main/App";

const csiData = ref({});
const amplitudeChart = ref(null);
const phaseChart = ref(null);
const phaseDiffChart = ref(null);
const polarChart = ref(null);

// 保存上一帧数据用于计算相位差
const prevFrame = ref(null);
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
    console.log("收到 CSI:", frame);
    csiData.value = frame;
    state.isConnected = true;

    let phaseDiff = null;
    if (prevFrame.value && prevFrame.value.phase && frame.phase) {
      phaseDiff = calculatePhaseDiff(prevFrame.value.phase, frame.phase);
    }

    // 实时绘制幅度图
    drawAmplitude(frame.amplitude);
    drawPhase(frame.phase);
    drawPhaseDiff(phaseDiff);
    drawPolar(frame.amplitude, frame.phase);

    // 保存当前帧作为上一帧
    prevFrame.value = {
      index: frame.index,
      phase: [...frame.phase],
      amplitude: [...frame.amplitude],
    };
  });
});

onUnmounted(() => {
  // 取消监听，防止内存泄漏
  if (unsubscribe) unsubscribe();
});

// 计算相位差（处理角度环绕问题）
function calculatePhaseDiff(prevPhase, currPhase) {
  const diff = [];
  const len = Math.min(prevPhase.length, currPhase.length);

  for (let i = 0; i < len; i++) {
    let delta = currPhase[i] - prevPhase[i];

    // 处理角度环绕：将差值归一化到 [-180, 180] 范围
    while (delta > 180) delta -= 360;
    while (delta < -180) delta += 360;

    diff.push(delta);
  }

  return diff;
}

function drawAmplitude(amplitude) {
  const canvas = amplitudeChart.value;
  if (!canvas || !amplitude) return;

  const ctx = canvas.getContext("2d");
  const w = canvas.width;
  const h = canvas.height;

  ctx.clearRect(0, 0, w, h);
  ctx.beginPath();
  ctx.strokeStyle = "#00ff00";

  const step = w / amplitude.length;
  const max = 35;

  amplitude.forEach((val, i) => {
    const x = i * step;
    const y = h - (val / max) * h * 0.8 - 10;
    if (i === 0) ctx.moveTo(x, y);
    else ctx.lineTo(x, y);
  });

  ctx.stroke();
  // 画网格线
  ctx.strokeStyle = "#333";
  ctx.lineWidth = 0.5;
  [0.25, 0.5, 0.75].forEach((ratio) => {
    ctx.beginPath();
    ctx.moveTo(0, h - 20 - ratio * (h - 40));
    ctx.lineTo(w, h - 20 - ratio * (h - 40));
    ctx.stroke();
  });
}
// 相位折线图
function drawPhase(phase) {
  const canvas = phaseChart.value;
  if (!canvas || !phase) return;

  const ctx = canvas.getContext("2d");
  const w = canvas.width;
  const h = canvas.height;

  ctx.fillStyle = "#1a1a2e";
  ctx.fillRect(0, 0, w, h);

  const step = w / phase.length;
  const centerY = h / 2;

  // 画0度基准线
  ctx.strokeStyle = "#444";
  ctx.lineWidth = 1;
  ctx.setLineDash([5, 5]);
  ctx.beginPath();
  ctx.moveTo(0, centerY);
  ctx.lineTo(w, centerY);
  ctx.stroke();
  ctx.setLineDash([]);

  // 画相位线
  ctx.beginPath();
  ctx.strokeStyle = "#ff6600";
  ctx.lineWidth = 1.5;

  phase.forEach((val, i) => {
    const x = i * step;
    // 角度范围 [-180, 180] 映射到画布
    const y = centerY - (val / 180) * (h / 2 - 20);
    if (i === 0) ctx.moveTo(x, y);
    else ctx.lineTo(x, y);
  });

  ctx.stroke();

  // 标注
  ctx.fillStyle = "#888";
  ctx.font = "12px sans-serif";
  ctx.fillText("180°", 5, 15);
  ctx.fillText("0°", 5, centerY + 4);
  ctx.fillText("-180°", 5, h - 5);
}

// 相位极坐标图
function drawPolar(amplitude, phase) {
  const canvas = polarChart.value;
  if (!canvas || !phase || !amplitude) return;

  const ctx = canvas.getContext("2d");
  const w = canvas.width;
  const h = canvas.height;
  const cx = w / 2;
  const cy = h / 2;
  const maxR = Math.min(w, h) / 2 - 30;

  // 清空
  ctx.fillStyle = "#1a1a2e";
  ctx.fillRect(0, 0, w, h);

  // 画同心圆
  ctx.strokeStyle = "#333";
  ctx.lineWidth = 1;
  [0.3, 0.6, 1.0].forEach((ratio) => {
    ctx.beginPath();
    ctx.arc(cx, cy, maxR * ratio, 0, Math.PI * 2);
    ctx.stroke();
  });

  // 画十字线
  ctx.beginPath();
  ctx.moveTo(cx - maxR, cy);
  ctx.lineTo(cx + maxR, cy);
  ctx.moveTo(cx, cy - maxR);
  ctx.lineTo(cx, cy + maxR);
  ctx.stroke();

  // 画相位点（颜色表示相位，半径表示幅度）
  const maxAmp = Math.max(...amplitude) || 1;

  phase.forEach((p, i) => {
    const amp = amplitude[i] || 0;
    const r = (amp / maxAmp) * maxR;
    // 角度转弧度，-90度调整让0度在上方
    const theta = ((p - 90) * Math.PI) / 180;

    const x = cx + r * Math.cos(theta);
    const y = cy + r * Math.sin(theta);

    // 色相映射：-180°=红，0°=绿，180°=蓝
    const hue = ((p + 180) / 360) * 240;
    ctx.fillStyle = `hsl(${hue}, 80%, 60%)`;
    ctx.beginPath();
    ctx.arc(x, y, 3, 0, Math.PI * 2);
    ctx.fill();
  });

  // 图例
  ctx.fillStyle = "#fff";
  ctx.font = "12px sans-serif";
  ctx.fillText("颜色: 相位 (-180°~180°)", 10, 20);
  ctx.fillText("半径: 幅度", 10, 35);
}

// 相位差图
function drawPhaseDiff(phaseDiff) {
  const canvas = phaseDiffChart.value;
  if (!canvas || !phaseDiff) {
    // 如果没有数据，显示提示
    if (canvas) {
      const ctx = canvas.getContext("2d");
      ctx.fillStyle = "#1a1a2e";
      ctx.fillRect(0, 0, canvas.width, canvas.height);
      ctx.fillStyle = "#666";
      ctx.font = "14px sans-serif";
      ctx.fillText("等待上一帧数据...", 10, canvas.height / 2);
    }
    return;
  }

  const ctx = canvas.getContext("2d");
  const w = canvas.width;
  const h = canvas.height;

  ctx.fillStyle = "#1a1a1e";
  ctx.fillRect(0, 0, w, h);

  const step = w / phaseDiff.length;
  const centerY = h / 2;

  // 画0度基准线（无变化线）
  ctx.strokeStyle = "#444";
  ctx.lineWidth = 1;
  ctx.setLineDash([5, 5]);
  ctx.beginPath();
  ctx.moveTo(0, centerY);
  ctx.lineTo(w, centerY);
  ctx.stroke();
  ctx.setLineDash([]);

  // 统计最大最小值用于颜色映射
  const maxDiff = Math.max(...phaseDiff.map(Math.abs));
  const avgDiff = phaseDiff.reduce((a, b) => a + Math.abs(b), 0) / phaseDiff.length;

  // 画相位差线
  ctx.beginPath();
  ctx.lineWidth = 2;

  phaseDiff.forEach((val, i) => {
    const x = i * step;
    // 相位差范围 [-180, 180] 映射到画布
    const y = centerY - (Math.abs(val) / 180) * (h / 2 - 20);

    // 根据差值大小设置颜色：小变化=绿色，大变化=红色
    const absVal = Math.abs(val);
    const intensity = Math.min(absVal / 45, 1); // 45度以上为最大强度
    const r = Math.floor(255 * intensity);
    const g = Math.floor(255 * (1 - intensity));
    ctx.strokeStyle = `rgb(${r}, ${g}, 100)`;

    if (i === 0) {
      ctx.moveTo(x, y);
    } else {
      ctx.lineTo(x, y);
      ctx.stroke();
      ctx.beginPath();
      ctx.moveTo(x, y);
    }
  });

  // 重绘为连续线条（使用渐变）
  const gradient = ctx.createLinearGradient(0, 0, w, 0);
  phaseDiff.forEach((val, i) => {
    const absVal = Math.abs(val);
    const intensity = Math.min(absVal / 90, 1);
    const r = Math.floor(255 * intensity);
    const g = Math.floor(255 * (1 - intensity * 0.5));
    const color = `rgb(${r}, ${g}, 50)`;
    gradient.addColorStop(i / phaseDiff.length, color);
  });

  ctx.beginPath();
  ctx.strokeStyle = gradient;
  ctx.lineWidth = 2;
  phaseDiff.forEach((val, i) => {
    const x = i * step;
    const y = centerY - (val / 180) * (h / 2 - 20);
    if (i === 0) ctx.moveTo(x, y);
    else ctx.lineTo(x, y);
  });
  ctx.stroke();

  // 标注
  ctx.fillStyle = "#888";
  ctx.font = "12px sans-serif";
  ctx.fillText("+180°", 5, 15);
  ctx.fillText("0° (无变化)", 5, centerY + 4);
  ctx.fillText("-180°", 5, h - 5);

  // 显示统计信息
  ctx.fillStyle = "#0f0";
  ctx.fillText(`最大变化: ${maxDiff.toFixed(1)}° | 平均变化: ${avgDiff.toFixed(1)}°`, w - 200, 15);
}
</script>

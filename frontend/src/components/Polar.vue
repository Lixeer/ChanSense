<template>
  <div class="polar flex items-center justify-center flex-col gap-1">
    <h3>相位极坐标 (Polar)</h3>
    <canvas ref="canvasRef" width="400" height="400"></canvas>
  </div>
</template>

<script setup lang="js">
import { ref, watch, onMounted } from "vue";

const props = defineProps({
  amplitude: {
    type: Array,
    default: () => [],
  },
  phase: {
    type: Array,
    default: () => [],
  },
});
const canvasRef = ref(null);

const draw = () => {
  const canvas = canvasRef.value;
  if (!canvas || !props.phase || !props.amplitude) return;

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
  const maxAmp = Math.max(...props.amplitude) || 1;

  props.phase.forEach((p, i) => {
    const amp = props.amplitude[i] || 0;
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
};

watch(
  () => {
    (props.phase, props.amplitude);
  },
  draw,
  { deep: true, flush: "post" },
);
onMounted(draw);
</script>

<style scoped lang="scss"></style>

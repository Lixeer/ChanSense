<template>
  <div class="amplitude-waterfall flex items-center justify-center flex-col gap-1">
    <h3>CSI 幅度数据瀑布图</h3>
    <canvas ref="canvasRef" :width="WIDTH" height="200"></canvas>
  </div>
</template>

<script setup lang="js">
import { ref, watch, onMounted, defineProps } from "vue";
const props = defineProps({
  amplitude: {
    type: Array,
    default: () => [],
  },
});
const canvasRef = ref(null);

const prevFrame = ref(null);
const MAX_HISTORY = 800;
const WIDTH = 800;
const MAX_AMPLITUDE = 75;
const AMPLITUDE_COUNT = ref(props.amplitude.length);
onMounted(() => {
  const canvas = canvasRef.value;
  if (!canvas) return;
  const ctx = canvas.getContext("2d");
  const w = canvas.width;
  const h = canvas.height;
  ctx.fillStyle = "#1a1a1e";
  ctx.fillRect(0, 0, w, h);
});
const draw = () => {
  const canvas = canvasRef.value;
  if (!canvas || !props.amplitude) return;
  if (!prevFrame.value) {
    prevFrame.value = [...props.amplitude];
    return;
  }
  const ctx = canvas.getContext("2d");
  const w = WIDTH;
  const h = canvas.height;

  const step = WIDTH / MAX_HISTORY;

  ctx.drawImage(canvas, 0, 0, w, h, -step, 0, w, h);
  ctx.clearRect(w - step, 0, step, h);
  ctx.fillStyle = "#1a1a1e";
  ctx.fillRect(w - step, 0, step, h);

  for (let i = 0; i < AMPLITUDE_COUNT.value; i++) {
    ctx.beginPath();
    const val = props.amplitude[i];
    const prevVal = prevFrame.value[i] ?? 0;
    const r = Math.floor((i / AMPLITUDE_COUNT.value) * 255);
    const g = Math.floor((1 - i / AMPLITUDE_COUNT.value) * 255);
    const color = `rgba(${r}, ${g}, 50, ${(val / MAX_AMPLITUDE) * 0.2})`;
    ctx.strokeStyle = color;
    ctx.lineWidth = 1.5;
    ctx.moveTo(w - step, h - (prevVal / MAX_AMPLITUDE) * h);
    ctx.lineTo(w, h - (val / MAX_AMPLITUDE) * h);
    ctx.stroke();
  }

  prevFrame.value = [...props.amplitude];
};

watch(
  () => props.amplitude,
  () => {
    draw();
  },
  { deep: true, flush: "post" },
);
</script>

<style scoped lang="scss"></style>

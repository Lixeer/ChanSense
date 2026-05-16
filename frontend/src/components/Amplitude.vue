<template>
  <div class="amplitude flex items-center justify-center flex-col gap-1">
    <h3 class="">幅度 (Amplitude)</h3>
    <canvas ref="canvasRef" width="800" height="200" class="border"></canvas>
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

const draw = () => {
  const canvas = canvasRef.value;
  if (!canvas || !props.amplitude) return;

  const ctx = canvas.getContext("2d");
  const w = canvas.width;
  const h = canvas.height;

  ctx.fillStyle = "#1a1a1e";
  ctx.fillRect(0, 0, w, h);

  ctx.beginPath();
  ctx.strokeStyle = "#0f0";
  ctx.lineWidth = 1;

  const step = w / props.amplitude.length;
  const max = 70;

  props.amplitude.forEach((val, i) => {
    const x = i * step;
    const y = h - (val / max) * (h - 40) - 20;
    if (i === 0) ctx.moveTo(x, y);
    else ctx.lineTo(x, y);
  });

  ctx.stroke();
  // 画网格线
  ctx.strokeStyle = "#444";
  ctx.lineWidth = 1;
  ctx.beginPath();
  [0.25, 0.5, 0.75].forEach((ratio) => {
    ctx.moveTo(0, h - 20 - ratio * (h - 40));
    ctx.lineTo(w, h - 20 - ratio * (h - 40));
  });
  ctx.stroke();
  ctx.beginPath();
  ctx.strokeStyle = "#666";
  ctx.lineWidth = 1.5;
  [0, 1].forEach((ratio) => {
    ctx.moveTo(0, h - 20 - ratio * (h - 40));
    ctx.lineTo(w, h - 20 - ratio * (h - 40));
  });
  ctx.stroke();
};

watch(() => props.amplitude, draw, { deep: true, flush: "post" });
onMounted(draw);
</script>

<style scoped lang="scss"></style>

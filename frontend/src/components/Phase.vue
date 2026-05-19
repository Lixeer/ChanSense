<template>
  <div class="phase flex items-center justify-center flex-col gap-1">
    <h3>相位 (Phase)</h3>
    <canvas ref="canvasRef" width="800" height="200" class="border"></canvas>
  </div>
</template>

<script setup lang="js">
import { ref, watch, onMounted } from "vue";
const props = defineProps({
  phase: {
    type: Array,
    default: () => [],
  },
});
const canvasRef = ref(null);

const draw = () => {
  const canvas = canvasRef.value;
  if (!canvas || !props.phase) return;

  const ctx = canvas.getContext("2d");
  const w = canvas.width;
  const h = canvas.height;

  ctx.fillStyle = "#1a1a2e";
  ctx.fillRect(0, 0, w, h);

  const step = w / props.phase.length;
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

  props.phase.forEach((val, i) => {
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
};

watch(() => props.phase, draw, { deep: true, flush: "post" });
onMounted(draw);
</script>

<style scoped lang="scss"></style>

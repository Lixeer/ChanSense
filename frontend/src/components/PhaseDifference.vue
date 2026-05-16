<template>
  <div class="phase-difference flex items-center justify-center flex-col gap-1">
    <h3>相位差 (Phase Difference) - 与上一帧对比</h3>
    <canvas ref="canvasRef" width="800" height="200" class="border"></canvas>
  </div>
</template>

<script setup lang="js">
import { ref, watch, onMounted, defineProps } from "vue";
const props = defineProps({
  phase: {
    type: Array,
    default: () => [],
  },
});
const canvasRef = ref(null);
const prevFrame = ref(null);
const phaseDiff = ref([]);
const diff = () => {
  if (!props.phase || props.phase.length === 0) return;
  if (!prevFrame.value) {
    prevFrame.value = [...props.phase];
    return;
  }
  phaseDiff.value = props.phase.map((val, i) => {
    const prevVal = prevFrame.value[i] ?? 0;
    let diff = val - prevVal;
    // 将差值限制在 [-180, 180] 范围内
    if (diff > 180) diff -= 360;
    else if (diff < -180) diff += 360;
    return diff;
  });
  prevFrame.value = [...props.phase];
};

const draw = () => {
  const canvas = canvasRef.value;
  if (!canvas || !phaseDiff.value) {
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

  const step = w / phaseDiff.value.length;
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
  const maxDiff = Math.max(...phaseDiff.value.map(Math.abs));
  const avgDiff = phaseDiff.value.reduce((a, b) => a + Math.abs(b), 0) / phaseDiff.value.length;

  // 画相位差线
  ctx.beginPath();
  ctx.lineWidth = 2;

  //   phaseDiff.value.forEach((val, i) => {
  //     const x = i * step;
  //     // 相位差范围 [-180, 180] 映射到画布
  //     const y = centerY - (val / 180) * (h / 2 - 20);

  //     // 根据差值大小设置颜色：小变化=绿色，大变化=红色
  //     const absVal = Math.abs(val);
  //     const intensity = Math.min(absVal / 135, 1); // 45度以上为最大强度
  //     const r = Math.floor(255 * intensity);
  //     const g = Math.floor(255 * (1 - intensity));
  //     ctx.strokeStyle = `rgb(${r}, ${g}, 100)`;

  //     if (i === 0) {
  //       ctx.moveTo(x, y);
  //     } else {
  //       ctx.lineTo(x, y);
  //       ctx.stroke();
  //       ctx.beginPath();
  //       ctx.moveTo(x, y);
  //     }
  //   });

  // 重绘为连续线条（使用渐变）
  const gradient = ctx.createLinearGradient(0, 0, w, 0);
  phaseDiff.value.forEach((val, i) => {
    const absVal = Math.abs(val);
    const intensity = Math.min(absVal / 135, 1);
    const r = Math.floor(255 * intensity);
    const g = Math.floor(255 * (1 - intensity * 0.6));
    const color = `rgb(${r}, ${g}, 50)`;
    gradient.addColorStop(i / phaseDiff.value.length, color);
  });

  ctx.beginPath();
  ctx.strokeStyle = gradient;
  ctx.lineWidth = 2;
  phaseDiff.value.forEach((val, i) => {
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
};

watch(
  () => props.phase,
  () => {
    diff();
    draw();
  },
  { deep: true, flush: "post" },
);
onMounted(draw);
</script>

<style scoped lang="scss"></style>

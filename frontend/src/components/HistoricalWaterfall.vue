<template>
  <div class="historical-waterfall w-full overflow-x-auto relative">
    <div class="" :style="{ width: width + 'px', height: '300px' }">
      <canvas
        ref="canvasRef"
        :width="width"
        height="300"
        class="absolute left-0 top-0 z-0"
        @mousedown="handleMouseDown"
      ></canvas>
      <div
        class="absolute left-0 top-0 h-full bg-blue-400 opacity-30 pointer-events-none z-1"
        :style="{ left: boxLeft + 'px', width: boxWidth + 'px' }"
      ></div>
    </div>
  </div>
</template>

<script setup lang="js">
import { useMouseInElement, clamp } from "@vueuse/core";
import { ref, watch, onMounted, computed } from "vue";
const props = defineProps({
  amplitude: {
    type: Array,
    default: () => [],
  },
  history: {
    type: Number,
    default: 200,
  },
});
const emit = defineEmits(["selected"]);
const canvasRef = ref(null);
const step = ref(4);
const MAX_AMPLITUDE = ref(75);
const width = computed(() => props.history * step.value);
const currentFrame = ref(0);

const prevFrame = ref(null);
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
    currentFrame.value = 1;
    prevFrame.value = [...props.amplitude];
    return;
  }
  const ctx = canvas.getContext("2d");
  const w = width.value;
  const h = canvas.height;

  for (let i = 0; i < props.amplitude.length; i++) {
    ctx.beginPath();
    const val = props.amplitude[i];
    const prevVal = prevFrame.value[i] ?? 0;
    const r = Math.floor((i / props.amplitude.length) * 255 * 0.8);
    const g = Math.floor((1 - i / props.amplitude.length) * 255);
    const color = `rgba(${r}, ${g}, 50, ${(val / MAX_AMPLITUDE.value) * 0.2})`;
    ctx.strokeStyle = color;
    ctx.lineWidth = 1.5;
    ctx.moveTo((currentFrame.value - 1) * step.value, h - (prevVal / MAX_AMPLITUDE.value) * h);
    ctx.lineTo(currentFrame.value * step.value, h - (val / MAX_AMPLITUDE.value) * h);
    ctx.stroke();
  }
  currentFrame.value++;
  prevFrame.value = [...props.amplitude];
};

watch(
  () => props.amplitude,
  () => {
    draw();
  },
  { deep: true },
);

const isSelecting = ref(false);
const startX = ref(0);
const currentX = ref(0);
const confirmedStart = ref(0);
const confirmedEnd = ref(0);
const { elementX, elementY, isOutside } = useMouseInElement(canvasRef);
// 鼠标按下：开始框选
const handleMouseDown = () => {
  // 如果滑出了画布范围，不触发
  if (isOutside.value) return;

  isSelecting.value = true;
  startX.value = clamp(elementX.value, 0, width.value);
  currentX.value = startX.value;
  // 监听全局的 mousemove 和 mouseup，确保鼠标滑出画布也能正常结束
  window.addEventListener("mousemove", handleMouseMove);
  window.addEventListener("mouseup", handleMouseUp);
};

// 鼠标移动：更新当前坐标
const handleMouseMove = () => {
  if (!isSelecting.value) return;
  currentX.value = clamp(elementX.value, 0, width.value);
};

// 鼠标松开：结束框选，抛出数据
const handleMouseUp = () => {
  if (!isSelecting.value) return;
  isSelecting.value = false;

  window.removeEventListener("mousemove", handleMouseMove);
  window.removeEventListener("mouseup", handleMouseUp);

  confirmedStart.value = startX.value;
  confirmedEnd.value = currentX.value;
  // 计算帧范围
  const startFrame = Math.floor(Math.min(confirmedStart.value, confirmedEnd.value) / step.value);
  const endFrame = Math.floor(Math.max(confirmedStart.value, confirmedEnd.value) / step.value);

  console.log("Selected frames:", startFrame, endFrame);
  emit("selected", {
    startIdx: startFrame,
    endIdx: endFrame,
    count: endFrame - startFrame,
  });
};

const boxLeft = computed(() => {
  if (isSelecting.value) {
    const clampedX = clamp(elementX.value, 0, width.value);
    return Math.min(startX.value, clampedX);
  }
  return Math.min(confirmedStart.value, confirmedEnd.value);
});

const boxWidth = computed(() => {
  if (isSelecting.value) {
    const clampedX = clamp(elementX.value, 0, width.value);
    return Math.abs(clampedX - startX.value);
  }
  return Math.abs(confirmedEnd.value - confirmedStart.value);
});
</script>

<style scoped lang="scss"></style>

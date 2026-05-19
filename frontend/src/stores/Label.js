import { defineStore } from "pinia";
import { ref } from "vue";

export const useLabelStore = defineStore("label", () => {
  const labels = ref([]);
  const activeLabel = ref(null);

  function addLabel(label) {
    labels.value.push(label);
    activeLabel.value = null;
  }
  function clearLabels() {
    labels.value = [];
    activeLabel.value = null;
  }
  function removeLabel(label) {
    const index = labels.value.indexOf(label);
    if (index !== -1) {
      labels.value.splice(index, 1);
      if (activeLabel.value === label) {
        activeLabel.value = null;
      }
    }
  }
  function setActiveLabel(label) {
    activeLabel.value = label;
  }
  return { labels, addLabel, clearLabels, removeLabel, activeLabel, setActiveLabel };
});

import {
  defineConfig,
  presetWind4,
  presetAttributify,
  presetIcons,
  transformerDirectives,
  transformerVariantGroup,
  presetTypography,
} from "unocss";

export default defineConfig({
  content: {
    pipeline: {
      include: [/\.(vue|svelte|[jt]sx|mdx?|astro|elm|php|phtml|html)($|\?)/, "src/**/*.{js,ts}"],
    },
  },
  presets: [
    presetWind4(), // UnoCSS 的 Tailwind CSS / Windi CSS 紧凑预设。
    presetAttributify({ prefix: "un-", prefixedOnly: false }), // 开启“属性化”支持，如 <div flex border="1">
    presetIcons({
      extraProperties: {
        display: "inline-block",
        "vertical-align": "middle",
        // ...
      },
    }), // 允许直接使用图标，如 class="i-mdi-home"
    presetTypography(),
  ],
  transformers: [
    transformerDirectives(), // 允许你在 <style> 里写 @apply
    transformerVariantGroup(), // 允许你写 hover:(bg-red text-white)
  ],
});

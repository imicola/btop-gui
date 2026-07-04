<script setup lang="ts">
defineProps<{ title: string; tone?: 'cyan' | 'green' | 'violet' | 'rose'; meta?: string }>()
</script>

<template>
  <section class="terminal-panel" :class="`tone-${tone || 'cyan'}`">
    <div class="panel-legend">
      <span class="corner">┤</span>{{ title }}<span v-if="meta" class="legend-meta">{{ meta }}</span><span class="corner">├</span>
    </div>
    <slot />
  </section>
</template>

<style scoped>
.terminal-panel {
  --tone: var(--cyan);
  position: relative;
  min-width: 0;
  min-height: 0;
  border: 1px solid color-mix(in srgb, var(--tone) 82%, var(--line));
  border-radius: 5px;
  background: linear-gradient(180deg, rgba(255,255,255,.018), transparent 24%), var(--panel);
  padding: 17px 10px 9px;
  overflow: visible;
}
.tone-green { --tone: var(--green); }
.tone-violet { --tone: var(--violet); }
.tone-rose { --tone: var(--rose); }
.panel-legend {
  position: absolute;
  top: 0;
  left: 9px;
  z-index: 4;
  display: flex;
  align-items: center;
  gap: 5px;
  max-width: calc(100% - 18px);
  height: 17px;
  padding: 0 5px;
  transform: translateY(-50%);
  background: var(--bg);
  color: var(--tone);
  font-size: 11.5px;
  font-weight: 600;
  line-height: 17px;
  letter-spacing: .055em;
  text-transform: uppercase;
  white-space: nowrap;
}
.corner { color: color-mix(in srgb, var(--tone) 55%, var(--muted)); }
.legend-meta {
  overflow: hidden;
  color: var(--muted);
  font-weight: 500;
  text-overflow: ellipsis;
  text-transform: none;
  letter-spacing: 0;
}
</style>

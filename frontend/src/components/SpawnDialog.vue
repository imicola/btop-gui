<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{ open: boolean; busy: boolean }>()
const emit = defineEmits<{ close: []; spawn: [name: string, args: string]; fork: [seconds: number] }>()
const name = ref('')
const args = ref('')
const seconds = ref(10)
watch(() => props.open, value => { if (value) { name.value = ''; args.value = ''; seconds.value = 10 } })
function submit() { if (name.value.trim()) emit('spawn', name.value.trim(), args.value) }
</script>

<template>
  <div v-if="open" class="overlay" @click.self="emit('close')" @keydown.esc="emit('close')">
    <div class="dialog" role="dialog" aria-modal="true" aria-label="启动进程">
      <div class="dialog-title"><span>┤ process laboratory ├</span><button @click="emit('close')">×</button></div>
      <p>直接创建进程，不经过 shell。参数支持引号与反斜线转义。</p>
      <label>PROGRAM</label>
      <input v-model="name" autofocus placeholder="/usr/bin/sleep" @keyup.enter="submit">
      <label>ARGUMENTS</label>
      <input v-model="args" placeholder="10  or  -m &quot;http.server&quot; 8080" @keyup.enter="submit">
      <button class="primary" :disabled="busy || !name.trim()" @click="submit">exec.Command 启动</button>
      <div class="fork-rule"><span>真实系统调用演示</span></div>
      <div class="fork-row">
        <div><b>fork() → execv() → wait4()</b><small>子进程执行 /bin/sleep，可在进程树中观察</small></div>
        <input v-model.number="seconds" type="number" min="1" max="30" aria-label="演示秒数">
        <button :disabled="busy || seconds < 1 || seconds > 30" @click="emit('fork', seconds)">运行 fork</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.overlay { position: fixed; inset: 0; z-index: 100; display: grid; place-items: center; background: rgba(9,10,18,.78); backdrop-filter: blur(3px); }
.dialog { width: min(510px, calc(100vw - 30px)); border: 1px solid var(--violet); border-radius: 5px; background: var(--panel); box-shadow: 0 22px 70px rgba(0,0,0,.5); padding: 18px; color: var(--text); font-family: Nunito, sans-serif; }
.dialog-title { display: flex; justify-content: space-between; margin-top: -27px; color: var(--violet); font: 700 13px var(--mono); }.dialog-title span { background: var(--panel); padding: 0 5px; }.dialog-title button { border: 0; background: var(--panel); color: var(--muted); font-size: 20px; cursor: pointer; }
p { color: var(--muted); font-size: 12px; margin: 15px 0; }
label { display: block; color: var(--cyan); font: 700 10px var(--mono); letter-spacing: .1em; margin: 9px 0 4px; }
input { width: 100%; border: 1px solid var(--line); border-radius: 3px; outline: none; background: var(--bg); color: var(--text); padding: 8px 10px; font: 12px var(--mono); } input:focus { border-color: var(--cyan); }
button { border: 1px solid var(--line); border-radius: 3px; background: transparent; color: var(--text-soft); padding: 7px 10px; font: 600 11px var(--mono); cursor: pointer; }.primary { width: 100%; margin-top: 12px; border-color: var(--cyan); color: var(--cyan); } button:disabled { opacity: .35; cursor: not-allowed; }
.fork-rule { display: flex; align-items: center; gap: 10px; margin: 17px 0 10px; color: var(--muted); font-size: 11px; }.fork-rule::before,.fork-rule::after { content: ''; flex: 1; border-top: 1px solid var(--line); }
.fork-row { display: grid; grid-template-columns: 1fr 58px auto; align-items: center; gap: 8px; }.fork-row b,.fork-row small { display: block; }.fork-row b { color: var(--green); font: 600 11px var(--mono); }.fork-row small { color: var(--muted); margin-top: 3px; font-size: 10px; }.fork-row input { text-align: center; }
</style>

<template>
  <div class="landing">
    <div class="landing-container">
      <div class="hero">
        <h1>mememory</h1>
        <p class="tagline">Persistent memory for AI agents.</p>
      </div>

      <div class="install">
        <div class="code-block" @click="copy" :title="copied ? 'Copied!' : 'Click to copy'">
          <span class="comment"># install</span><br>
          <span class="prompt">$ </span>go install github.com/scott-walker/mememory@latest<br>
          <br>
          <span class="comment"># setup — one command, everything works</span><br>
          <span class="prompt">$ </span>mememory setup
        </div>
      </div>

      <div class="features">
        <div class="feature">
          <div class="feature-title">Pluggable</div>
          <div class="feature-desc">Ollama, OpenAI, or any embedding provider. Local Postgres or cloud. Your choice.</div>
        </div>
        <div class="feature">
          <div class="feature-title">MCP native</div>
          <div class="feature-desc">Works with Claude Code, Claude Desktop, and any MCP-compatible client.</div>
        </div>
      </div>

      <div class="links">
        <a href="https://github.com/scott-walker/mememory">GitHub</a>
        <a :href="withBase('/guide/getting-started')">Get Started</a>
        <a :href="withBase('/reference/mcp-tools')">API Reference</a>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { withBase } from 'vitepress'
const copied = ref(false)
function copy() {
  navigator.clipboard.writeText('go install github.com/scott-walker/mememory@latest')
  copied.value = true
  setTimeout(() => copied.value = false, 2000)
}
</script>

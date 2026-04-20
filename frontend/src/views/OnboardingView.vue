<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useTheme } from '../composables/useTheme'
import { useProjectsStore } from '../stores/projects'

defineOptions({ name: 'OnboardingView' })

const router = useRouter()
const projectsStore = useProjectsStore()
const { theme, setTheme } = useTheme()

type StepId = 'welcome' | 'project' | 'integrations' | 'theme' | 'done'

const steps: { id: StepId; label: string }[] = [
  { id: 'welcome',      label: 'Welcome' },
  { id: 'project',      label: 'Project' },
  { id: 'integrations', label: 'Integrations' },
  { id: 'theme',        label: 'Theme' },
]

const stepIdx = ref(0)
const currentStep = computed<StepId>(() => steps[stepIdx.value]?.id ?? 'done')

const projectPath = ref('')
const integrations = ref<{ docker: boolean; gitlab: boolean; makefile: boolean }>({
  docker: true,
  gitlab: false,
  makefile: true,
})

function next() {
  if (stepIdx.value < steps.length - 1) stepIdx.value++
  else finish()
}

function prev() {
  if (stepIdx.value > 0) stepIdx.value--
}

function finish() {
  try { localStorage.setItem('devhub.onboarded', '1') } catch {}
  router.push('/')
}

function skip() {
  try { localStorage.setItem('devhub.onboarded', '1') } catch {}
  router.push('/')
}

const stepProgress = computed(() => steps.map((s, i) => ({
  ...s,
  done: i < stepIdx.value,
  cur:  i === stepIdx.value,
})))
</script>

<template>
  <div class="onboarding-view">
    <div class="wiz">
      <div class="wiz-stage" :class="{ splash: currentStep === 'welcome' }">
        <!-- Splash header (welcome only) -->
        <template v-if="currentStep === 'welcome'">
          <div class="splash-big"></div>
          <h1 class="splash-title">Welcome to <em>DevHub</em></h1>
          <p class="splash-sub">
            One hub for git, docker, issues and the terminal — running entirely on your machine.
            This setup takes about 60 seconds.
          </p>
          <div class="splash-actions">
            <button class="btn lg" @click="skip">Skip — show me the app</button>
            <button class="btn primary lg" @click="next">Get started →</button>
          </div>
          <div class="splash-meta">
            <span><span class="bullet ok">●</span> Local-first · no telemetry</span>
            <span><span class="bullet warn">●</span> MIT licensed</span>
            <span><span class="bullet info">●</span> v0.3.0</span>
          </div>
        </template>

        <!-- Wizard header (steps 2+) -->
        <template v-else>
          <div class="wiz-header">
            <div class="wiz-mark"></div>
            <div class="steps">
              <template v-for="(s, i) in stepProgress.slice(1)" :key="s.id">
                <div class="step" :class="{ done: s.done, cur: s.cur }">
                  <span class="num">
                    <template v-if="s.done">✓</template>
                    <template v-else>{{ i + 2 }}</template>
                  </span>
                  {{ s.label }}
                </div>
                <span v-if="i < stepProgress.length - 2" class="step-line" :class="{ done: s.done }"></span>
              </template>
            </div>
            <button class="btn ghost sm" @click="skip">Skip</button>
          </div>

          <div class="wiz-body">
            <!-- Step 2: Project -->
            <template v-if="currentStep === 'project'">
              <h2>Pick a project folder</h2>
              <p class="lede">Tell DevHub where your code lives. We'll scan for git, Docker Compose, and a Makefile to detect features.</p>
              <div class="path-input">
                <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" style="color: var(--fg-3)">
                  <path d="M2 4h5l2 2h5v8H2z"/>
                </svg>
                <input v-model="projectPath" placeholder="~/code/my-project" spellcheck="false" />
                <button class="btn sm">Browse</button>
              </div>
              <div class="picker">
                <button
                  v-for="p in projectsStore.projects.slice(0, 4)"
                  :key="p.name"
                  class="pick"
                  :class="{ sel: projectPath === p.path }"
                  @click="projectPath = p.path"
                >
                  <span class="icon">
                    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
                      <path d="M2 3h5l2 2h5v9H2z"/>
                    </svg>
                  </span>
                  <div>
                    <div class="t">{{ p.name }}</div>
                    <div class="d">{{ p.path }}</div>
                  </div>
                </button>
              </div>
            </template>

            <!-- Step 3: Integrations -->
            <template v-else-if="currentStep === 'integrations'">
              <h2>Detect tools</h2>
              <p class="lede">DevHub auto-detects what's available. Toggle any you don't want.</p>
              <div class="detect-list">
                <div class="detect-row">
                  <span class="tick"><span class="bullet ok">●</span></span>
                  <div>
                    <div style="font-weight: 600">Git</div>
                    <div class="info">working tree at the project root</div>
                  </div>
                  <button class="tgl on" disabled aria-label="Always on"></button>
                </div>
                <div class="detect-row">
                  <span class="tick"><span class="bullet ok">●</span></span>
                  <div>
                    <div style="font-weight: 600">Docker Compose</div>
                    <div class="info">docker-compose.yml detected</div>
                  </div>
                  <button class="tgl" :class="{ on: integrations.docker }" aria-label="Docker" @click="integrations.docker = !integrations.docker"></button>
                </div>
                <div class="detect-row">
                  <span class="tick"><span class="bullet ok">●</span></span>
                  <div>
                    <div style="font-weight: 600">Makefile</div>
                    <div class="info">6 targets exposed as quick actions</div>
                  </div>
                  <button class="tgl" :class="{ on: integrations.makefile }" aria-label="Makefile" @click="integrations.makefile = !integrations.makefile"></button>
                </div>
                <div class="detect-row">
                  <span class="tick"><span class="bullet warn">●</span></span>
                  <div>
                    <div style="font-weight: 600">GitLab</div>
                    <div class="info">no token configured · optional</div>
                  </div>
                  <button class="tgl" :class="{ on: integrations.gitlab }" aria-label="GitLab" @click="integrations.gitlab = !integrations.gitlab"></button>
                </div>
              </div>
            </template>

            <!-- Step 4: Theme -->
            <template v-else-if="currentStep === 'theme'">
              <h2>Pick your theme</h2>
              <p class="lede">DevHub ships in two flavours. You can switch any time from the sidebar.</p>
              <div class="picker">
                <button class="pick" :class="{ sel: theme === 'dark' }" @click="setTheme('dark')">
                  <span class="icon">
                    <svg viewBox="0 0 16 16" fill="currentColor"><path d="M13 9.5A5.5 5.5 0 017.5 4a5.5 5.5 0 01.3-1.8A6 6 0 1013.8 9.2a5.5 5.5 0 01-.8.3z"/></svg>
                  </span>
                  <div>
                    <div class="t">Warm dark</div>
                    <div class="d">Default. Easy on the eyes after sundown.</div>
                  </div>
                </button>
                <button class="pick" :class="{ sel: theme === 'light' }" @click="setTheme('light')">
                  <span class="icon">
                    <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="8" cy="8" r="3"/><path d="M8 1v2M8 13v2M1 8h2M13 8h2M3 3l1.5 1.5M11.5 11.5L13 13M3 13l1.5-1.5M11.5 4.5L13 3"/></svg>
                  </span>
                  <div>
                    <div class="t">Warm paper</div>
                    <div class="d">Bright, high-contrast. Great for daytime.</div>
                  </div>
                </button>
              </div>
              <div class="finish-cards">
                <div class="finish-card">
                  <div class="k">Tip</div>
                  <div class="t">Press <span class="kbd">Ctrl+K</span></div>
                  <div class="d">Open the command palette from anywhere.</div>
                </div>
                <div class="finish-card">
                  <div class="k">Tip</div>
                  <div class="t">Press <span class="kbd">?</span></div>
                  <div class="d">View all keyboard shortcuts.</div>
                </div>
                <div class="finish-card">
                  <div class="k">Tip</div>
                  <div class="t">Press <span class="kbd">Ctrl+`</span></div>
                  <div class="d">Toggle the bottom terminal.</div>
                </div>
              </div>
            </template>
          </div>

          <div class="wiz-footer">
            <button class="btn ghost" :disabled="stepIdx === 0" @click="prev">← Back</button>
            <div class="footer-dots">
              <span
                v-for="(s, i) in steps"
                :key="s.id"
                class="footer-dot"
                :class="{ cur: i === stepIdx }"
              ></span>
            </div>
            <button class="btn primary" @click="next">
              {{ stepIdx === steps.length - 1 ? 'Open DevHub →' : 'Next →' }}
            </button>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.onboarding-view {
  width: 100%;
  min-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--s6) var(--s4);
}

.wiz { width: 100%; max-width: 880px; }

.wiz-stage {
  padding: 48px 56px 40px;
  background: var(--bg-1);
  border: 1px solid var(--line);
  border-radius: var(--r3);
  box-shadow: var(--shadow-3);
  min-height: 520px;
  display: flex;
  flex-direction: column;
}
.wiz-stage.splash {
  text-align: center;
  align-items: center;
  padding: 64px 40px 48px;
}

/* Splash */
.splash-big {
  width: 104px; height: 104px;
  border-radius: 28px;
  position: relative;
  background: conic-gradient(from 210deg, oklch(72% 0.13 68), oklch(68% 0.13 150), oklch(72% 0.11 230), oklch(72% 0.13 68));
  box-shadow: 0 20px 80px oklch(72% 0.13 68 / .35);
}
.splash-big::after {
  content: ""; position: absolute; inset: 10px;
  border-radius: 18px; background: var(--bg-1);
}
.splash-big::before {
  content: ""; position: absolute; left: 36px; top: 30px;
  width: 28px; height: 44px;
  border-left: 4px solid var(--fg);
  border-bottom: 4px solid var(--fg);
  transform: rotate(-45deg);
  border-radius: 2px;
}
.splash-title {
  font-size: 40px;
  font-weight: 700;
  letter-spacing: -0.03em;
  margin: 24px 0 10px;
  color: var(--fg);
}
.splash-title em { font-style: normal; color: var(--accent); }
.splash-sub {
  color: var(--fg-3);
  font-size: 15px;
  max-width: 480px;
  margin: 0 auto;
}
.splash-actions {
  display: flex;
  gap: 10px;
  justify-content: center;
  margin-top: 32px;
}
.splash-meta {
  display: flex;
  gap: 24px;
  justify-content: center;
  margin-top: 32px;
  color: var(--fg-3);
  font-size: 12px;
}
.bullet { display: inline-block; }
.bullet.ok   { color: var(--ok); }
.bullet.warn { color: var(--warn); }
.bullet.info { color: var(--info); }

/* Wizard chrome */
.wiz-header {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 28px;
}
.wiz-mark {
  width: 44px; height: 44px;
  border-radius: 12px;
  position: relative;
  background: conic-gradient(from 210deg, oklch(72% 0.13 68), oklch(68% 0.13 150), oklch(72% 0.11 230), oklch(72% 0.13 68));
  box-shadow: 0 6px 24px oklch(72% 0.13 68 / .35);
  flex-shrink: 0;
}
.wiz-mark::after {
  content: ""; position: absolute; inset: 5px;
  border-radius: 7px; background: var(--bg-1);
}
.wiz-mark::before {
  content: ""; position: absolute;
  left: 16px; top: 12px;
  width: 12px; height: 20px;
  border-left: 3px solid var(--fg);
  border-bottom: 3px solid var(--fg);
  transform: rotate(-45deg);
}

.steps {
  display: flex;
  align-items: center;
  gap: 0;
  flex: 1;
  margin: 0 24px;
}
.step {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--fg-3);
  font-size: 12px;
  white-space: nowrap;
}
.step.done { color: var(--ok); }
.step.cur  { color: var(--fg); }
.step .num {
  width: 22px; height: 22px;
  border-radius: 50%;
  border: 1px solid var(--line);
  background: var(--bg-2);
  display: flex; align-items: center; justify-content: center;
  font-size: 11px;
  font-family: var(--mono);
}
.step.done .num {
  background: var(--ok-2);
  border-color: var(--ok);
  color: var(--ok);
}
.step.cur .num {
  background: var(--accent);
  border-color: var(--accent);
  color: var(--accent-ink);
  box-shadow: 0 0 0 4px var(--accent-2);
}
.step-line {
  flex: 1;
  height: 1px;
  background: var(--line);
  margin: 0 8px;
}
.step-line.done { background: var(--ok); }

.wiz-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.wiz-body h2 {
  font-size: 28px;
  font-weight: 700;
  letter-spacing: -0.02em;
  margin: 0;
  color: var(--fg);
}
.wiz-body .lede {
  color: var(--fg-3);
  font-size: 14px;
  max-width: 520px;
  margin: 0;
}

.path-input {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 14px;
  background: var(--bg-2);
  border: 1px solid var(--line);
  border-radius: var(--r2);
  font-family: var(--mono);
  font-size: 13.5px;
}
.path-input input {
  flex: 1;
  background: transparent;
  border: 0;
  outline: 0;
  color: var(--fg);
  font-family: var(--mono);
  font-size: 13.5px;
}

.picker {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}
.pick {
  padding: 16px;
  border: 1px solid var(--line);
  border-radius: var(--r2);
  background: var(--bg-2);
  cursor: pointer;
  display: flex;
  gap: 12px;
  align-items: flex-start;
  transition: border-color var(--t-fast), background var(--t-fast);
  font-family: var(--ui);
  text-align: left;
}
.pick:hover { border-color: var(--fg-3); }
.pick.sel { border-color: var(--accent); background: var(--accent-2); }
.pick .icon {
  width: 32px; height: 32px;
  border-radius: 8px;
  background: var(--bg-1);
  border: 1px solid var(--line);
  display: flex; align-items: center; justify-content: center;
  color: var(--accent);
  flex-shrink: 0;
}
.pick .t {
  font-weight: 600;
  font-size: 13.5px;
  color: var(--fg);
}
.pick .d {
  font-size: 12px;
  color: var(--fg-3);
  margin-top: 3px;
}

.detect-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.detect-row {
  display: grid;
  grid-template-columns: 24px 1fr auto;
  gap: 12px;
  align-items: center;
  padding: 12px 14px;
  background: var(--bg-2);
  border: 1px solid var(--line);
  border-radius: var(--r2);
  font-size: 13.5px;
}
.detect-row .info {
  font-family: var(--mono);
  font-size: 12px;
  color: var(--fg-3);
}

.tgl {
  position: relative;
  width: 36px;
  height: 20px;
  border-radius: var(--r-pill);
  background: var(--bg-3);
  border: 1px solid var(--line);
  cursor: pointer;
  transition: background var(--t-fast), border-color var(--t-fast);
  padding: 0;
}
.tgl::after {
  content: "";
  position: absolute;
  left: 2px; top: 2px;
  width: 14px; height: 14px;
  border-radius: 50%;
  background: var(--fg-2);
  transition: all var(--t-fast);
}
.tgl.on { background: var(--accent); border-color: var(--accent); }
.tgl.on::after { left: 18px; background: var(--accent-ink); }

.finish-cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  margin-top: 12px;
}
.finish-card {
  padding: 16px;
  border: 1px solid var(--line);
  border-radius: var(--r2);
  background: var(--bg-2);
  text-align: left;
  cursor: pointer;
  transition: border-color var(--t-fast);
}
.finish-card:hover { border-color: var(--accent); }
.finish-card .t {
  font-weight: 600;
  color: var(--fg);
  margin-top: 8px;
  font-size: 13.5px;
}
.finish-card .d {
  color: var(--fg-3);
  font-size: 12px;
  margin-top: 4px;
}
.finish-card .k {
  font-family: var(--mono);
  color: var(--accent);
  font-size: 11px;
  letter-spacing: .08em;
  text-transform: uppercase;
}

.wiz-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 24px;
  margin-top: auto;
  border-top: 1px solid var(--line-soft);
}
.footer-dots {
  display: flex;
  gap: 6px;
  align-items: center;
}
.footer-dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  background: var(--line);
  transition: width var(--t-fast), background var(--t-fast), border-radius var(--t-fast);
}
.footer-dot.cur {
  background: var(--accent);
  width: 18px;
  border-radius: 3px;
}
</style>

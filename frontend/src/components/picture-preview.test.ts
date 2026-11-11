import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { runInNewContext } from 'node:vm'
import { compileScript, parse } from '@vue/compiler-sfc'
import ts from 'typescript'
import { createSSRApp, defineComponent, h } from 'vue'
import { renderToString } from 'vue/server-renderer'

const require = createRequire(import.meta.url)
const { descriptor } = parse(readFileSync(new URL('./picture-preview.vue', import.meta.url), 'utf8'))
const compiled = ts.transpileModule(compileScript(descriptor, { id: 'preview-test', inlineTemplate: true }).content,
  { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText
const exports: any = {}
runInNewContext(compiled, { exports, require })
async function render(reviewImg: boolean, reviewUrl: string) {
  const viewers: any[] = []
  const app = createSSRApp(exports.default, { reviewImg, reviewUrl })
  app.component('t-image-viewer', defineComponent({ props: ['visible', 'images'], setup(p) {
    viewers.push(p); return () => h('viewer')
  } }))
  await renderToString(app)
  return viewers
}
test('closed or empty previews do not mount a viewer with a failing default thumbnail', async () => {
  assert.equal((await render(false, '')).length, 0)
  assert.equal((await render(false, 'blob:old')).length, 0)
  assert.equal((await render(true, '')).length, 0)
})
test('an explicitly opened image keeps its exact prepared URL', async () => {
  const viewers = await render(true, 'blob:authorized-image')
  assert.equal(viewers.length, 1)
  assert.deepEqual(Array.from(viewers[0].images), ['blob:authorized-image'])
  assert.equal(viewers[0].visible, true)
})

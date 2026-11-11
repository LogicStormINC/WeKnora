import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { runInNewContext } from 'node:vm'
import test from 'node:test'
const source = readFileSync(new URL('./PdfSourceViewer.vue', import.meta.url), 'utf8')
const onScroll = source.match(/function onScroll\(\) \{[\s\S]*?\n\}/)[0]
const jump = source.match(/function scrollToPageFraction\(page: number, fraction: number\) \{[\s\S]*?\n\}/)[0].replace('page: number, fraction: number', 'page, fraction')
test('whole-page citations align to the page top and retain its toolbar number after native scroll rounding', () => {
 let top = 0
 const box = { clientHeight: 700, get scrollTop() { return top }, set scrollTop(v) { top = Math.floor(v) } }
 const pages = new Map([[6,{ offsetTop: 0, offsetHeight: 1000 }],[7,{ offsetTop:1012.3, offsetHeight:1000 }]])
 const currentPage = { value: 6 }
 const context = { scroller:{value:box}, pageElements:pages, currentPage, PAGE_GAP:12 }
 runInNewContext(onScroll+'\n'+jump+'\nscrollToPageFraction(7, 0); onScroll();',context)
 assert.equal(currentPage.value,7)
 assert.ok(top >= 1012 && top <= 1013, 'whole page must begin at the viewport top')
})

// 清空 Vite 产物，但保留 dist/.gitkeep。
//
// dist 由后端 //go:embed all:dist 引用，全新克隆时靠 .gitkeep 让目录存在，
// 因此不能用 vite 的 emptyOutDir（它会连 .gitkeep 一起删掉，导致每次构建
// 都在 git 里留下一条 "D internal/web/dist/.gitkeep"）。
import { readdir, rm } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const dist = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'internal', 'web', 'dist')

for (const entry of await readdir(dist)) {
  if (entry === '.gitkeep') continue
  await rm(join(dist, entry), { recursive: true, force: true })
}

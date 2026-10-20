// Vite empties dist/ on every build, which would delete dist/.gitkeep and show
// up as a spurious change in git. Recreate it after each build so the folder
// stays tracked: `go:embed web/dist` needs it to exist on a fresh clone.
import { mkdirSync, writeFileSync } from 'node:fs'

const note = `This folder holds the compiled web client.

\`pnpm build\` in \`web/\` writes index.html and assets/ here, and the Go binary
embeds them at compile time. The file you are reading only exists so that
\`go:embed web/dist\` resolves on a fresh clone, before the first web build.
`

mkdirSync('dist', { recursive: true })
writeFileSync('dist/.gitkeep', note)

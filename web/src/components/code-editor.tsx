import { json } from '@codemirror/lang-json'
import { EditorView } from '@codemirror/view'
import CodeMirror from '@uiw/react-codemirror'
import { useMemo } from 'react'
import { useTheme } from '@/lib/theme'

type Props = {
  value: string
  onChange?: (value: string) => void
  readOnly?: boolean
  language?: 'json' | 'text'
  minHeight?: string
  maxHeight?: string
  placeholder?: string
  ariaLabel?: string
}

/**
 * CodeMirror wrapped so it inherits the workspace palette. Both themes are
 * defined here rather than per-call so every editor in the app stays identical.
 */
export function CodeEditor({
  value,
  onChange,
  readOnly = false,
  language = 'json',
  minHeight = '8rem',
  maxHeight,
  placeholder,
  ariaLabel,
}: Props) {
  const { theme } = useTheme()
  const dark = theme === 'dark'

  const extensions = useMemo(() => {
    const base = [EditorView.lineWrapping]
    return language === 'json' ? [...base, json()] : base
  }, [language])

  const palette = dark
    ? {
        bg: 'transparent',
        fg: '#d8dde4',
        caret: '#8fb6ff',
        selection: '#2a3a52',
        gutter: '#6b7684',
        active: '#1e242c',
      }
    : {
        bg: 'transparent',
        fg: '#23272e',
        caret: '#2b5fd9',
        selection: '#cfe0ff',
        gutter: '#8d95a1',
        active: '#eef1f5',
      }

  const themeExt = useMemo(
    () =>
      EditorView.theme(
        {
          '&': { backgroundColor: palette.bg, color: palette.fg, height: '100%' },
          '.cm-content': { caretColor: palette.caret, padding: '10px 0' },
          '.cm-cursor, .cm-dropCursor': { borderLeftColor: palette.caret },
          '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection':
            { backgroundColor: palette.selection },
          '.cm-gutters': { backgroundColor: 'transparent', color: palette.gutter },
          '.cm-activeLine': { backgroundColor: palette.active },
          '.cm-activeLineGutter': { backgroundColor: palette.active },
          '.cm-placeholder': { color: palette.gutter },
        },
        { dark },
      ),
    [dark, palette.active, palette.bg, palette.caret, palette.fg, palette.gutter, palette.selection],
  )

  return (
    <div
      className="overflow-hidden rounded-md border border-border/70 bg-background/40"
      aria-label={ariaLabel}
    >
      <CodeMirror
        value={value}
        onChange={onChange}
        readOnly={readOnly}
        editable={!readOnly}
        extensions={[...extensions, themeExt]}
        placeholder={placeholder}
        height="auto"
        minHeight={minHeight}
        maxHeight={maxHeight}
        style={{ fontSize: 12.5 }}
        basicSetup={{
          lineNumbers: true,
          foldGutter: false,
          highlightActiveLine: !readOnly,
          highlightActiveLineGutter: !readOnly,
          autocompletion: false,
          bracketMatching: true,
          closeBrackets: true,
          tabSize: 2,
        }}
      />
    </div>
  )
}

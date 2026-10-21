import { PlusIcon, Trash2Icon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from 'cn'
import type { KV } from '@/lib/types'
import { useT } from '@/lib/i18n'

type Props = {
  rows: KV[]
  onChange: (rows: KV[]) => void
  keyLabel?: string
  valueLabel?: string
  emptyHint: string
  addLabel?: string
  /** 'query' or 'header' — decides placeholder examples. */
  kind: 'query' | 'header' | 'form'
}

const placeholders = {
  query: { key: 'limit', value: '10' },
  header: { key: 'Authorization', value: 'Bearer …' },
  form: { key: 'field', value: 'value' },
}

/**
 * A spreadsheet-like editor for key/value rows. Rows are always visible so the
 * shape of a request never has to be discovered by clicking.
 */
export function KVEditor({
  rows,
  onChange,
  keyLabel,
  valueLabel,
  emptyHint,
  addLabel,
  kind,
}: Props) {
  const t = useT()
  const resolvedKeyLabel = keyLabel ?? t('kv.key')
  const resolvedValueLabel = valueLabel ?? t('kv.value')
  const resolvedAddLabel = addLabel ?? t('common.addRow')

  const update = (id: string, patch: Partial<KV>) => {
    onChange(rows.map((row) => (row.id === id ? { ...row, ...patch } : row)))
  }

  const remove = (id: string) => {
    onChange(rows.filter((row) => row.id !== id))
  }

  const add = () => {
    onChange([
      ...rows,
      { id: crypto.randomUUID(), key: '', value: '', enabled: true, description: '' },
    ])
  }

  return (
    <div className="flex flex-col">
      <div className="grid grid-cols-[1.5rem_minmax(0,1fr)_minmax(0,1.4fr)_2rem] items-center gap-2 border-b border-border/60 px-1 pb-1.5">
        <span />
        <span className="eyebrow">{resolvedKeyLabel}</span>
        <span className="eyebrow">{resolvedValueLabel}</span>
        <span />
      </div>

      {rows.length === 0 ? (
        <button
          type="button"
          onClick={add}
          className="group mt-2 flex h-16 w-full items-center justify-center rounded-md border border-dashed border-border/70 text-xs text-muted-foreground transition-colors hover:border-border hover:bg-accent/40 hover:text-foreground"
        >
          <PlusIcon className="mr-1.5 size-3.5" />
          {emptyHint}
        </button>
      ) : (
        <div className="mt-1 flex flex-col">
          {rows.map((row) => (
            <div
              key={row.id}
              className={cn(
                'group grid grid-cols-[1.5rem_minmax(0,1fr)_minmax(0,1.4fr)_2rem] items-center gap-2 rounded px-1 py-0.5',
                !row.enabled && 'opacity-45',
              )}
            >
              <input
                type="checkbox"
                checked={row.enabled}
                onChange={(e) => update(row.id, { enabled: e.target.checked })}
                aria-label={t('kv.includeRow', { name: row.key || 'row' })}
                className="size-3.5 cursor-pointer accent-[var(--ring)]"
              />
              <Input
                value={row.key}
                onChange={(e) => update(row.id, { key: e.target.value })}
                placeholder={placeholders[kind].key}
                spellCheck={false}
                className="mono h-7 border-transparent bg-transparent px-1.5 text-xs shadow-none hover:border-border/60 focus-visible:border-border"
              />
              <Input
                value={row.value}
                onChange={(e) => update(row.id, { value: e.target.value })}
                placeholder={placeholders[kind].value}
                spellCheck={false}
                className="mono h-7 border-transparent bg-transparent px-1.5 text-xs shadow-none hover:border-border/60 focus-visible:border-border"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon-xs"
                onClick={() => remove(row.id)}
                aria-label={t('kv.removeRow')}
                className="opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
              >
                <Trash2Icon className="size-3.5" />
              </Button>
            </div>
          ))}
        </div>
      )}

      {rows.length > 0 && (
        <Button
          type="button"
          variant="ghost"
          size="xs"
          onClick={add}
          className="mt-1.5 w-fit text-muted-foreground"
        >
          <PlusIcon className="size-3.5" />
          {resolvedAddLabel}
        </Button>
      )}
    </div>
  )
}

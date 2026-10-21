import { PlusIcon, SaveIcon, Trash2Icon } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { useConfirm } from '@/components/confirm-dialog'
import { api } from '@/lib/api'
import { useWorkspace } from '@/lib/store'
import type { EnvVar, Environment } from '@/lib/types'
import { useT, tNow } from '@/lib/i18n'
import { toast } from 'sonner'
import { cn } from 'cn'

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

const blankVar = (): EnvVar => ({
  id: crypto.randomUUID(),
  key: '',
  value: '',
  enabled: true,
  secret: false,
})

export function EnvironmentDialog({ open, onOpenChange }: Props) {
  const t = useT()
  const { environments, activeEnvId, setWorkspace, collections, requests } = useWorkspace()
  const [selected, setSelected] = useState<string | null>(null)
  const [name, setName] = useState('')
  const [vars, setVars] = useState<EnvVar[]>([])
  const [saving, setSaving] = useState(false)
  const { confirm, dialog: confirmDialog } = useConfirm()

  const current: Environment | undefined = environments.find((e) => e.id === selected)

  useEffect(() => {
    if (!open) return
    const first = selected ?? activeEnvId ?? environments[0]?.id ?? null
    setSelected(first)
    // Only re-seed the selection when the dialog opens; environments change as
    // the user edits, and following them here would fight the form.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  useEffect(() => {
    if (!open) return
    if (current) {
      setName(current.name)
      setVars(current.variables.map((v) => ({ ...v })))
    } else {
      setName('')
      setVars([])
    }
    // Reload the form only when a different environment is picked, never while
    // the user is typing into it.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [current?.id, open])

  const reload = async (nextActive?: string) => {
    const res = await api.environments()
    setWorkspace({
      collections,
      requests,
      environments: res.items,
      activeEnvId: nextActive ?? res.active,
    })
    return res
  }

  const save = async () => {
    setSaving(true)
    try {
      if (current) {
        await api.updateEnvironment({ ...current, name: name || tNow('env.untitled'), variables: vars })
        toast.success(tNow('toast.environmentSaved'))
      } else {
        const created = await api.createEnvironment(name || tNow('env.untitled'), vars)
        setSelected(created.id)
        toast.success(tNow('toast.environmentCreated'))
      }
      await reload()
    } finally {
      setSaving(false)
    }
  }

  const create = () => {
    setSelected(null)
    setName(tNow('env.newEnvironment'))
    setVars([blankVar()])
  }

  const remove = async () => {
    if (!current) return
    if (
      !(await confirm({
        title: tNow('confirm.deleteEnvironment.title', { name: current.name }),
        description:
          current.id === activeEnvId
            ? tNow('confirm.deleteEnvironment.descActive')
            : tNow('confirm.deleteEnvironment.desc'),
        confirmLabel: tNow('confirm.deleteEnvironment.confirm'),
      }))
    )
      return
    await api.deleteEnvironment(current.id)
    setSelected(null)
    await reload(current.id === activeEnvId ? '' : undefined)
    toast.success(tNow('toast.environmentDeleted'))
  }

  const activate = async () => {
    if (!current) return
    await api.activateEnvironment(current.id)
    await reload(current.id)
    toast.success(tNow('toast.usingEnvironment', { name: current.name }))
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="gap-0 overflow-hidden p-0 sm:max-w-3xl">
        <DialogHeader className="flex-row items-center gap-2 space-y-0 border-b border-border px-4 py-3 pr-12">
          <DialogTitle className="text-sm">{t('env.title')}</DialogTitle>
          <DialogDescription className="text-[11px]">
            {t('env.subtitle')}
          </DialogDescription>
          <Button variant="outline" size="xs" onClick={create} className="ml-auto">
            <PlusIcon /> {t('env.new')}
          </Button>
        </DialogHeader>

        <div className="grid h-[420px] grid-cols-[190px_minmax(0,1fr)]">
          <div className="overflow-y-auto border-r border-border bg-muted/30 p-2">
            {environments.length === 0 && selected === null && (
              <p className="px-2 py-3 text-[11px] text-muted-foreground">
                {t('env.noEnvironments')}
              </p>
            )}
            {environments.map((env) => (
              <button
                key={env.id}
                type="button"
                onClick={() => setSelected(env.id)}
                className={cn(
                  'flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-left text-[12px] transition-colors',
                  selected === env.id
                    ? 'bg-background text-foreground shadow-sm'
                    : 'text-muted-foreground hover:bg-accent/60 hover:text-foreground',
                )}
              >
                <span className="min-w-0 flex-1 truncate">{env.name}</span>
                {activeEnvId === env.id && (
                  <span className="mono rounded bg-ok/15 px-1 text-[9px] text-ok">{t('env.activeBadge')}</span>
                )}
              </button>
            ))}
          </div>

          <div className="flex min-h-0 flex-col">
            <div className="flex items-center gap-2 border-b border-border px-3 py-2">
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t('env.namePlaceholder')}
                className="h-7 max-w-[220px] text-xs font-medium"
              />
              {current && activeEnvId !== current.id && (
                <Button variant="outline" size="xs" onClick={() => void activate()}>
                  {t('env.useThis')}
                </Button>
              )}
              <div className="ml-auto flex gap-1">
                {current && (
                  <Button variant="ghost" size="icon-xs" onClick={() => void remove()} aria-label={t('env.aria.delete')}>
                    <Trash2Icon className="size-3.5 text-bad" />
                  </Button>
                )}
                <Button size="xs" onClick={() => void save()} disabled={saving}>
                  <SaveIcon /> {t('common.save')}
                </Button>
              </div>
            </div>

            <div className="min-h-0 flex-1 overflow-y-auto p-3">
              {vars.length === 0 ? (
                <p className="py-6 text-center text-[11px] text-muted-foreground">
                  {t('env.noVariables')}
                </p>
              ) : (
                <div className="flex flex-col gap-1">
                  {vars.map((v, i) => (
                    <div key={v.id} className="grid grid-cols-[1.5rem_minmax(0,1fr)_minmax(0,1.4fr)_3rem] items-center gap-1.5">
                      <input
                        type="checkbox"
                        checked={v.enabled}
                        onChange={(e) =>
                          setVars(vars.map((x, j) => (j === i ? { ...x, enabled: e.target.checked } : x)))
                        }
                        aria-label={t('env.aria.enableVariable', { name: v.key || t('env.aria.enableVariableFallback') })}
                        className="size-3.5 accent-[var(--ring)]"
                      />
                      <Input
                        value={v.key}
                        onChange={(e) =>
                          setVars(vars.map((x, j) => (j === i ? { ...x, key: e.target.value } : x)))
                        }
                        placeholder="baseUrl"
                        className="mono h-7 text-[11.5px]"
                      />
                      <Input
                        type={v.secret ? 'password' : 'text'}
                        value={v.value}
                        onChange={(e) =>
                          setVars(vars.map((x, j) => (j === i ? { ...x, value: e.target.value } : x)))
                        }
                        placeholder={t('env.baseUrlPlaceholder')}
                        className="mono h-7 text-[11.5px]"
                      />
                      <Button
                        variant="ghost"
                        size="icon-xs"
                        aria-label={t('env.aria.removeVariable')}
                        onClick={() => setVars(vars.filter((_, j) => j !== i))}
                      >
                        <Trash2Icon className="size-3" />
                      </Button>
                    </div>
                  ))}
                </div>
              )}
              <Button
                variant="ghost"
                size="xs"
                className="mt-2 text-muted-foreground"
                onClick={() => setVars([...vars, blankVar()])}
              >
                <PlusIcon /> {t('env.addVariable')}
              </Button>
            </div>
          </div>
        </div>

        {/*
          This dialog's content uses p-0 so the panes can reach the edges. The
          footer is meant to be a full-bleed band, but DialogFooter's default
          -mx-4 -mb-4 would then push it outside the card (and overflow-hidden
          would clip the Done button). Zero the negative margins here.
        */}
        <DialogFooter className="mx-0 mb-0 border-t border-border px-4 py-2.5">
          <p className="mr-auto self-center text-[10.5px] text-muted-foreground">
            {t('env.footer')}
          </p>
          <Button variant="outline" size="sm" onClick={() => onOpenChange(false)}>
            {t('common.done')}
          </Button>
        </DialogFooter>
      </DialogContent>

      {confirmDialog}
    </Dialog>
  )
}

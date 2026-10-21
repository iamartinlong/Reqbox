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
import { useWorkspace } from '@/lib/store'
import { useT } from '@/lib/i18n'

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

/** Defaults applied to every send. Kept small on purpose: three knobs, no tabs. */
export function RunSettingsDialog({ open, onOpenChange }: Props) {
  const t = useT()
  const { options, setOptions } = useWorkspace()

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle className="text-sm">{t('run.title')}</DialogTitle>
          <DialogDescription className="text-[11px]">
            {t('run.desc')}
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-3">
          <label className="flex items-center justify-between gap-3">
            <span className="text-[12px]">{t('run.timeout')}</span>
            <span className="flex items-center gap-1.5">
              <Input
                type="number"
                min={1}
                max={600}
                value={options.timeoutSeconds}
                onChange={(e) => setOptions({ timeoutSeconds: Number(e.target.value) || 30 })}
                className="mono h-7 w-20 text-xs"
              />
              <span className="text-[11px] text-muted-foreground">{t('run.seconds')}</span>
            </span>
          </label>

          <label className="flex items-center justify-between gap-3">
            <span className="text-[12px]">{t('run.followRedirects')}</span>
            <input
              type="checkbox"
              checked={options.followRedirects}
              onChange={(e) => setOptions({ followRedirects: e.target.checked })}
              className="size-3.5 accent-[var(--ring)]"
            />
          </label>

          <label className="flex items-start justify-between gap-3">
            <span>
              <span className="block text-[12px]">{t('run.verifyTls')}</span>
              <span className="block text-[10.5px] text-muted-foreground">
                {t('run.verifyTlsHint')}
              </span>
            </span>
            <input
              type="checkbox"
              checked={options.verifyTls}
              onChange={(e) => setOptions({ verifyTls: e.target.checked })}
              className="mt-0.5 size-3.5 accent-[var(--ring)]"
            />
          </label>
        </div>

        <DialogFooter>
          <Button size="sm" onClick={() => onOpenChange(false)}>
            {t('common.done')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

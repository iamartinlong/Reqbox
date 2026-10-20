import { useCallback, useEffect, useRef, useState } from 'react'
import { TriangleAlertIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { cn } from 'cn'

export type ConfirmOptions = {
  title: string
  description?: string
  confirmLabel?: string
  cancelLabel?: string
  /** Renders the confirm button in the destructive style. Defaults to true. */
  destructive?: boolean
}

/**
 * Replaces window.confirm. The native dialog is unstyled, carries the app name
 * in its title bar, and is not available in every embedded browser context.
 *
 * Usage: `if (await confirmDestructive({ title: 'Delete "abc"?' })) { ... }`
 * The promise resolves false on cancel, Escape, or clicking the backdrop.
 */
export function ConfirmDialog({
  open,
  options,
  onResolve,
}: {
  open: boolean
  options: ConfirmOptions | null
  onResolve: (ok: boolean) => void
}) {
  if (!options) return null

  const destructive = options.destructive ?? true

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onResolve(false)}>
      <DialogContent
        className="sm:max-w-sm"
        onOpenAutoFocus={(e) => {
          // Focus the cancel button so an accidental Enter cannot delete data.
          e.preventDefault()
          const el = document.getElementById('confirm-dialog-cancel')
          el?.focus()
        }}
      >
        <DialogHeader>
          <DialogTitle className="flex items-start gap-2 text-sm">
            {destructive && (
              <TriangleAlertIcon className="mt-px size-4 shrink-0 text-bad" aria-hidden />
            )}
            <span>{options.title}</span>
          </DialogTitle>
          {options.description && (
            <DialogDescription className="text-[11.5px] leading-relaxed">
              {options.description}
            </DialogDescription>
          )}
        </DialogHeader>

        <DialogFooter className="mx-0 mb-0 border-t-0 bg-transparent p-0 pt-1">
          <Button
            id="confirm-dialog-cancel"
            variant="outline"
            size="sm"
            type="button"
            onClick={() => onResolve(false)}
          >
            {options.cancelLabel ?? 'Cancel'}
          </Button>
          <Button
            size="sm"
            type="button"
            variant={destructive ? 'destructive' : 'default'}
            onClick={() => onResolve(true)}
            className={cn(destructive && 'border-transparent')}
          >
            {options.confirmLabel ?? 'Delete'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/**
 * Hosts the dialog and returns an imperative `confirm` function, so call sites
 * read like the native one: `if (!(await confirm({...}))) return`.
 */
export function useConfirm() {
  // Content and visibility are separate so the dialog can animate out: the
  // options stay mounted while `open` goes false.
  const [options, setOptions] = useState<ConfirmOptions | null>(null)
  const [open, setOpen] = useState(false)
  const confirmRef = useRef<((ok: boolean) => void) | null>(null)

  const confirm = useCallback((next: ConfirmOptions) => {
    return new Promise<boolean>((resolve) => {
      // A second call while one is open resolves the previous one as cancelled.
      confirmRef.current?.(false)
      confirmRef.current = resolve
      setOptions(next)
      setOpen(true)
    })
  }, [])

  const resolve = useCallback((ok: boolean) => {
    const current = confirmRef.current
    confirmRef.current = null
    setOpen(false)
    current?.(ok)
  }, [])

  // Resolve any open dialog on unmount so callers never await forever.
  useEffect(() => {
    return () => {
      confirmRef.current?.(false)
      confirmRef.current = null
    }
  }, [])

  return {
    confirm,
    dialog: <ConfirmDialog open={open} options={options} onResolve={resolve} />,
  }
}

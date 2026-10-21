import { useEffect, useRef, useState } from 'react'
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
import { useT } from '@/lib/i18n'

export type PromptField = {
  name: string
  label: string
  value: string
  placeholder?: string
  hint?: string
}

type Props = {
  open: boolean
  title: string
  description?: string
  fields: PromptField[]
  confirmLabel?: string
  onCancel: () => void
  onSubmit: (values: Record<string, string>) => void
}

/**
 * Replaces window.prompt. The native prompt is unavailable in some embedded
 * browser contexts (it throws, so the click appears to do nothing) and cannot be
 * styled or validated. This dialog always works and can show more than one field.
 */
export function PromptDialog({
  open,
  title,
  description,
  fields,
  confirmLabel,
  onCancel,
  onSubmit,
}: Props) {
  const t = useT()
  const [values, setValues] = useState<Record<string, string>>({})
  const firstRef = useRef<HTMLInputElement>(null)

  // Re-seed whenever the dialog opens so stale values never leak between uses.
  useEffect(() => {
    if (!open) return
    const seed: Record<string, string> = {}
    for (const f of fields) seed[f.name] = f.value
    setValues(seed)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  const submit = () => {
    onSubmit(values)
  }

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onCancel()}>
      <DialogContent
        className="sm:max-w-sm"
        onOpenAutoFocus={(e) => {
          e.preventDefault()
          firstRef.current?.focus()
          firstRef.current?.select()
        }}
      >
        <DialogHeader>
          <DialogTitle className="text-sm">{title}</DialogTitle>
          {description && (
            <DialogDescription className="text-[11.5px]">{description}</DialogDescription>
          )}
        </DialogHeader>

        <form
          className="grid gap-3"
          onSubmit={(e) => {
            e.preventDefault()
            submit()
          }}
        >
          {fields.map((field, i) => (
            <label key={field.name} className="grid gap-1">
              <span className="eyebrow">{field.label}</span>
              <Input
                ref={i === 0 ? firstRef : undefined}
                value={values[field.name] ?? ''}
                onChange={(e) => setValues((v) => ({ ...v, [field.name]: e.target.value }))}
                placeholder={field.placeholder}
                className="text-xs"
                autoComplete="off"
              />
              {field.hint && (
                <span className="text-[10.5px] text-muted-foreground/80">{field.hint}</span>
              )}
            </label>
          ))}

          {/* A submit button keeps Enter working without a window-level listener. */}
          <button type="submit" className="hidden" aria-hidden tabIndex={-1} />
        </form>

        {/*
          DialogFooter ships with -mx-4 -mb-4 p-4 because it is designed as a
          full-bleed band inside DialogContent's default p-4. Resetting the
          negative margins keeps the buttons inside the card and lets the
          dialog's own padding provide the space below them.
        */}
        <DialogFooter className="mx-0 mb-0 border-t-0 bg-transparent p-0 pt-1">
          <Button variant="outline" size="sm" type="button" onClick={onCancel}>
            {t('common.cancel')}
          </Button>
          <Button size="sm" type="button" onClick={submit}>
            {confirmLabel ?? t('common.save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

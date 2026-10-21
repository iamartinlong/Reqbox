import { TerminalIcon } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { CodeEditor } from '@/components/code-editor'
import { api } from '@/lib/api'
import { useWorkspace } from '@/lib/store'
import { tNow, useT } from '@/lib/i18n'
import { toast } from 'sonner'

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function CurlDialog({ open, onOpenChange }: Props) {
  const t = useT()
  const { draft, activeEnvId } = useWorkspace()
  const [curl, setCurl] = useState('')

  useEffect(() => {
    if (!open || !draft) return
    api
      .curl({ request: draft, environmentId: activeEnvId })
      .then((res) => setCurl(res.curl))
      .catch(() => toast.error(tNow('toast.cannotBuildCurl')))
  }, [open, draft, activeEnvId])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-1.5 text-sm">
            <TerminalIcon className="size-4" /> {t('curl.title')}
          </DialogTitle>
          <DialogDescription className="text-[11px]">
            {t('curl.subtitle')}
          </DialogDescription>
        </DialogHeader>
        <CodeEditor value={curl} readOnly language="text" minHeight="10rem" ariaLabel="curl command" />
        <div className="flex justify-end gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              void navigator.clipboard.writeText(curl)
              toast.success(tNow('common.copied'))
            }}
          >
            {t('common.copy')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}

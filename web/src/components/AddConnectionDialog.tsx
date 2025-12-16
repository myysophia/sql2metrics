import { useState } from 'react'
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
import { Label } from '@/components/ui/label'

interface AddConnectionDialogProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    onConfirm: (name: string) => void
    title: string
    description?: string
    placeholder?: string
}

export function AddConnectionDialog({
    open,
    onOpenChange,
    onConfirm,
    title,
    description,
    placeholder = '请输入连接名称',
}: AddConnectionDialogProps) {
    const [name, setName] = useState('')

    const handleConfirm = () => {
        if (name.trim()) {
            onConfirm(name.trim())
            setName('')
            onOpenChange(false)
        }
    }

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[425px]">
                <DialogHeader>
                    <DialogTitle>{title}</DialogTitle>
                    {description && <DialogDescription>{description}</DialogDescription>}
                </DialogHeader>
                <div className="grid gap-4 py-4">
                    <div className="grid grid-cols-4 items-center gap-4">
                        <Label htmlFor="name" className="text-right">
                            名称
                        </Label>
                        <Input
                            id="name"
                            value={name}
                            onChange={(e) => setName(e.target.value)}
                            placeholder={placeholder}
                            className="col-span-3"
                            autoFocus
                            onKeyDown={(e) => {
                                if (e.key === 'Enter') handleConfirm()
                            }}
                        />
                    </div>
                </div>
                <DialogFooter>
                    <Button variant="outline" onClick={() => onOpenChange(false)}>
                        取消
                    </Button>
                    <Button onClick={handleConfirm} disabled={!name.trim()}>
                        确认
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    )
}

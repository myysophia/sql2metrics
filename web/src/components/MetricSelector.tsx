import { ChevronDown } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

interface MetricSelectorProps {
  availableMetrics: string[]
  selectedMetrics: string[]
  onChange: (metrics: string[]) => void
}

export default function MetricSelector({
  availableMetrics,
  selectedMetrics,
  onChange,
}: MetricSelectorProps) {
  const handleToggle = (metric: string) => {
    if (selectedMetrics.includes(metric)) {
      onChange(selectedMetrics.filter((m) => m !== metric))
    } else {
      onChange([...selectedMetrics, metric])
    }
  }

  const handleClear = () => {
    onChange([])
  }

  const handleSelectAll = () => {
    onChange(availableMetrics)
  }

  return (
    <div className="flex items-center gap-2">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" className="min-w-[200px] justify-between">
            {selectedMetrics.length > 0
              ? `已选择 ${selectedMetrics.length} 个指标`
              : '选择指标'}
            <ChevronDown className="ml-2 h-4 w-4 opacity-50" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent className="w-56" align="start">
          <DropdownMenuLabel>选择指标（多选）</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <div className="max-h-[300px] overflow-y-auto">
            {availableMetrics.length === 0 ? (
              <div className="px-2 py-4 text-sm text-muted-foreground text-center">
                无可用指标
              </div>
            ) : (
              availableMetrics.map((metric) => (
                <DropdownMenuCheckboxItem
                  key={metric}
                  checked={selectedMetrics.includes(metric)}
                  onCheckedChange={() => handleToggle(metric)}
                >
                  <span className="truncate">{metric}</span>
                </DropdownMenuCheckboxItem>
              ))
            )}
          </div>
          <DropdownMenuSeparator />
          <div className="flex flex-col gap-1 p-2">
            <Button
              variant="ghost"
              size="sm"
              className="h-8 text-xs"
              onClick={handleSelectAll}
              disabled={selectedMetrics.length === availableMetrics.length}
            >
              全选
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="h-8 text-xs"
              onClick={handleClear}
              disabled={selectedMetrics.length === 0}
            >
              清空
            </Button>
          </div>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}

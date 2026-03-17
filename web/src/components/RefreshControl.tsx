import { RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { RefreshInterval } from '../types/timeseries'

interface RefreshControlProps {
  autoRefresh: boolean
  onAutoRefreshChange: (enabled: boolean) => void
  refreshInterval: RefreshInterval
  onRefreshIntervalChange: (interval: RefreshInterval) => void
  onManualRefresh: () => void
  isRefreshing?: boolean
}

const refreshIntervalOptions: { value: RefreshInterval; label: string }[] = [
  { value: 30, label: '30 秒' },
  { value: 60, label: '1 分钟' },
  { value: 300, label: '5 分钟' },
]

export default function RefreshControl({
  autoRefresh,
  onAutoRefreshChange,
  refreshInterval,
  onRefreshIntervalChange,
  onManualRefresh,
  isRefreshing = false,
}: RefreshControlProps) {
  return (
    <div className="flex items-center gap-4">
      <div className="flex items-center gap-2">
        <Switch
          id="auto-refresh"
          checked={autoRefresh}
          onCheckedChange={onAutoRefreshChange}
        />
        <label
          htmlFor="auto-refresh"
          className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
        >
          自动刷新
        </label>
      </div>

      {autoRefresh && (
        <Select
          value={refreshInterval.toString()}
          onValueChange={(val) => onRefreshIntervalChange(Number(val) as RefreshInterval)}
        >
          <SelectTrigger className="w-[120px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {refreshIntervalOptions.map((option) => (
              <SelectItem key={option.value} value={option.value.toString()}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}

      <Button
        variant="outline"
        size="sm"
        onClick={onManualRefresh}
        disabled={isRefreshing}
      >
        <RefreshCw className={`mr-2 h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`} />
        刷新
      </Button>
    </div>
  )
}

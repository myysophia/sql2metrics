import { Button } from '@/components/ui/button'
import type { TimeRangePreset, TimeRangeOption } from '../types/timeseries'

interface TimeRangeSelectorProps {
  value: TimeRangePreset
  onChange: (range: TimeRangePreset) => void
}

const timeRangeOptions: TimeRangeOption[] = [
  { value: '5m', label: '5 分钟' },
  { value: '15m', label: '15 分钟' },
  { value: '1h', label: '1 小时' },
  { value: '6h', label: '6 小时' },
  { value: '24h', label: '1 天' },
  { value: '7d', label: '7 天' },
]

export default function TimeRangeSelector({ value, onChange }: TimeRangeSelectorProps) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-sm text-muted-foreground">时间范围:</span>
      <div className="flex gap-1">
        {timeRangeOptions.map((option) => (
          <Button
            key={option.value}
            variant={value === option.value ? 'default' : 'outline'}
            size="sm"
            onClick={() => onChange(option.value)}
          >
            {option.label}
          </Button>
        ))}
      </div>
    </div>
  )
}

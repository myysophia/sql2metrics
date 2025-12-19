import { useState } from 'react'
import { ChevronRight, ChevronDown } from 'lucide-react'

interface JsonFieldSelectorProps {
    data: unknown
    onSelect: (path: string) => void
    basePath?: string
}

interface JsonNodeProps {
    name: string
    value: unknown
    path: string
    onSelect: (path: string) => void
}

function isNumeric(value: unknown): boolean {
    return typeof value === 'number' ||
        (typeof value === 'string' && !isNaN(parseFloat(value)))
}

function JsonNode({ name, value, path, onSelect }: JsonNodeProps) {
    const [expanded, setExpanded] = useState(true)

    const isObject = value !== null && typeof value === 'object'
    const isArray = Array.isArray(value)
    const isSelectableValue = isNumeric(value)

    const handleClick = () => {
        if (isSelectableValue) {
            onSelect(path)
        }
    }

    const toggleExpand = (e: React.MouseEvent) => {
        e.stopPropagation()
        setExpanded(!expanded)
    }

    if (isObject) {
        const entries = isArray
            ? (value as unknown[]).map((v, i) => [`[${i}]`, v] as [string, unknown])
            : Object.entries(value as Record<string, unknown>)

        return (
            <div className="ml-2">
                <div
                    className="flex items-center gap-1 py-0.5 cursor-pointer hover:bg-muted rounded"
                    onClick={toggleExpand}
                >
                    {expanded ? (
                        <ChevronDown className="h-3 w-3 text-muted-foreground" />
                    ) : (
                        <ChevronRight className="h-3 w-3 text-muted-foreground" />
                    )}
                    <span className="text-blue-600">{name}</span>
                    <span className="text-muted-foreground">
                        {isArray ? `[${(value as unknown[]).length}]` : `{${entries.length}}`}
                    </span>
                </div>
                {expanded && (
                    <div className="ml-2 border-l border-muted pl-2">
                        {entries.map(([key, val]) => {
                            const childPath = isArray
                                ? `${path}${key}`
                                : path ? `${path}.${key}` : key
                            return (
                                <JsonNode
                                    key={key}
                                    name={key}
                                    value={val}
                                    path={childPath}
                                    onSelect={onSelect}
                                />
                            )
                        })}
                    </div>
                )}
            </div>
        )
    }

    return (
        <div
            className={`ml-2 py-0.5 flex items-center gap-2 ${isSelectableValue ? 'cursor-pointer hover:bg-primary/10 rounded px-1' : ''}`}
            onClick={handleClick}
        >
            <span className="text-purple-600">{name}:</span>
            <span className={isSelectableValue ? 'text-green-600 font-medium' : 'text-muted-foreground'}>
                {String(value)}
            </span>
            {isSelectableValue && (
                <span className="text-xs text-muted-foreground ml-auto">← 点击选择</span>
            )}
        </div>
    )
}

export function JsonFieldSelector({ data, onSelect, basePath = '' }: JsonFieldSelectorProps) {
    if (data === null || data === undefined) {
        return <div className="text-muted-foreground">无数据</div>
    }

    const isArray = Array.isArray(data)
    const isObject = typeof data === 'object'

    if (!isObject) {
        return (
            <div
                className="cursor-pointer hover:bg-primary/10 rounded px-1 py-0.5"
                onClick={() => onSelect(basePath || '')}
            >
                <span className="text-green-600">{String(data)}</span>
                <span className="text-xs text-muted-foreground ml-2">← 点击选择</span>
            </div>
        )
    }

    const entries = isArray
        ? (data as unknown[]).map((v, i) => [`[${i}]`, v] as [string, unknown])
        : Object.entries(data as Record<string, unknown>)

    return (
        <div>
            {entries.map(([key, val]) => {
                const path = isArray ? `${basePath}${key}` : basePath ? `${basePath}.${key}` : key
                return (
                    <JsonNode
                        key={key}
                        name={key}
                        value={val}
                        path={path}
                        onSelect={onSelect}
                    />
                )
            })}
        </div>
    )
}

import { useCallback } from 'react'
import CodeMirror from '@uiw/react-codemirror'
import { json, jsonParseLinter } from '@codemirror/lang-json'
import { linter } from '@codemirror/lint'
import { Button, Space, message } from 'antd'
import { CopyOutlined, FormatPainterOutlined } from '@ant-design/icons'

interface JsonEditorProps {
  value: string
  onChange: (value: string) => void
}

export default function JsonEditor({ value, onChange }: JsonEditorProps) {
  const format = useCallback(() => {
    try {
      onChange(JSON.stringify(JSON.parse(value), null, 2))
      message.success('已格式化')
    } catch {
      message.error('JSON 格式错误')
    }
  }, [value, onChange])

  const copy = useCallback(() => {
    navigator.clipboard.writeText(value).then(
      () => message.success('已复制'),
      () => message.error('复制失败'),
    )
  }, [value])

  return (
    <div>
      <Space style={{ marginBottom: 8 }}>
        <Button size="small" icon={<FormatPainterOutlined />} onClick={format}>格式化</Button>
        <Button size="small" icon={<CopyOutlined />} onClick={copy}>复制</Button>
      </Space>
      <CodeMirror
        value={value}
        onChange={onChange}
        extensions={[json(), linter(jsonParseLinter())]}
        theme="light"
        height="400px"
        style={{ fontSize: 12, borderRadius: 6, border: '1px solid #d9d9d9' }}
        basicSetup={{
          lineNumbers: true,
          foldGutter: true,
          bracketMatching: true,
          autocompletion: false,
          indentOnInput: true,
        }}
      />
    </div>
  )
}
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Steps, Card, Form, Input, InputNumber, Button, Typography, message, Alert, Space, type FormInstance } from 'antd'
import { DatabaseOutlined, KeyOutlined, UserOutlined, CheckCircleOutlined, ReloadOutlined } from '@ant-design/icons'
import { getSetupStatus, testDb, genSecrets, finishSetup, type DbReq, type FinishReq } from '../../api/setup'

const { Title, Text } = Typography

export default function SetupWizard() {
  const navigate = useNavigate()
  const [step, setStep] = useState(0)
  const [dbForm] = Form.useForm()
  const [secretForm] = Form.useForm()
  const [adminForm] = Form.useForm()
  const [testing, setTesting] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [dbOk, setDbOk] = useState(false)
  // 分步表单随步骤切换卸载，validateFields 取不到已卸载字段的值；每步通过时缓存
  const [saved, setSaved] = useState<Record<string, unknown>>({})

  // 已初始化则直接跳登录
  useEffect(() => {
    getSetupStatus().then((s) => {
      if (s.initialized) navigate('/login', { replace: true })
    })
  }, [navigate])

  // 进入密钥步骤时预填随机值
  useEffect(() => {
    if (step === 1) {
      genSecrets().then((s) => {
        secretForm.setFieldsValue({ jwt_secret: s.jwt_secret, encryption_key: s.encryption_key })
      })
    }
  }, [step, secretForm])

  const onTestDb = async () => {
    const values = await dbForm.validateFields()
    setTesting(true)
    try {
      const res = await testDb(values as DbReq)
      if (res.ok) {
        setDbOk(true)
        setSaved((s) => ({ ...s, ...values }))
        message.success(res.message)
      } else {
        setDbOk(false)
        message.error(res.message)
      }
    } finally {
      setTesting(false)
    }
  }

  // 进入下一步时缓存当前步骤表单值
  const cacheAndGo = async (form: FormInstance, next: number) => {
    const values = await form.validateFields()
    setSaved((s) => ({ ...s, ...values }))
    setStep(next)
  }

  const onFinish = async () => {
    const admin = await adminForm.validateFields()
    setSubmitting(true)
    try {
      await finishSetup({ ...(saved as unknown as FinishReq), ...admin })
      setStep(3)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div
      style={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'linear-gradient(135deg, #1890ff 0%, #096dd9 100%)',
      }}
    >
      <Card style={{ width: 560, boxShadow: '0 8px 32px rgba(0,0,0,.15)' }}>
        <Title level={3} style={{ textAlign: 'center', marginBottom: 8 }}>
          系统初始化
        </Title>
        <Text type="secondary" style={{ display: 'block', textAlign: 'center', marginBottom: 24 }}>
          首次使用请完成以下配置
        </Text>

        <Steps
          current={step}
          items={[
            { title: '数据库', icon: <DatabaseOutlined /> },
            { title: '安全密钥', icon: <KeyOutlined /> },
            { title: '管理员', icon: <UserOutlined /> },
            { title: '完成', icon: <CheckCircleOutlined /> },
          ]}
          style={{ marginBottom: 32 }}
        />

        {step === 0 && (
          <Form form={dbForm} layout="vertical" initialValues={{ host: '127.0.0.1', port: 3306, user: 'root', password: '', name: 'feedback' }}>
            <Space.Compact block>
              <Form.Item name="host" rules={[{ required: true, message: '请输入数据库地址' }]} style={{ flexGrow: 1 }}>
                <Input placeholder="数据库地址" />
              </Form.Item>
              <Form.Item name="port" rules={[{ required: true, message: '请输入端口' }]} style={{ width: 110 }}>
                <InputNumber min={1} max={65535} style={{ width: '100%' }} placeholder="端口" />
              </Form.Item>
            </Space.Compact>
            <Form.Item name="user" rules={[{ required: true, message: '请输入用户名' }]}>
              <Input placeholder="用户名" />
            </Form.Item>
            <Form.Item name="password">
              <Input.Password placeholder="密码（可为空）" />
            </Form.Item>
            <Form.Item name="name" rules={[{ required: true, message: '请输入数据库名' }]}>
              <Input placeholder="数据库名（需已创建）" />
            </Form.Item>
            {dbOk && <Alert type="success" showIcon message="连接成功，表结构已就绪" style={{ marginBottom: 16 }} />}
            <Button type="primary" block loading={testing} onClick={onTestDb} disabled={dbOk}>
              {dbOk ? '已通过' : '测试连接'}
            </Button>
            <Button type="primary" block style={{ marginTop: 12 }} disabled={!dbOk} onClick={() => setStep(1)}>
              下一步
            </Button>
          </Form>
        )}

        {step === 1 && (
          <Form form={secretForm} layout="vertical">
            <Alert
              type="info"
              showIcon
              message="已自动生成安全密钥，可直接使用；也可替换为您自己的值"
              style={{ marginBottom: 16 }}
            />
            <Form.Item
              name="jwt_secret"
              label="JWT 签名密钥"
              rules={[{ required: true, min: 16, message: '至少 16 个字符' }]}
              extra="用于登录令牌签名，请妥善保管，泄露需重新生成"
            >
              <Input.TextArea rows={2} />
            </Form.Item>
            <Form.Item
              name="encryption_key"
              label="数据加密密钥"
              rules={[{ required: true, message: '请输入 44 位 base64 密钥' }]}
              extra="用于加密敏感数据（如 LLM API Key），更换后旧密文将无法解密"
            >
              <Input.TextArea rows={2} />
            </Form.Item>
            <Space style={{ width: '100%' }} direction="vertical" size={12}>
              <Button block onClick={() => genSecrets().then((s) => secretForm.setFieldsValue(s))} icon={<ReloadOutlined />}>
                重新生成
              </Button>
              <Button type="primary" block onClick={() => cacheAndGo(secretForm, 2)}>
                下一步
              </Button>
              <Button type="link" block onClick={() => setStep(0)}>
                上一步
              </Button>
            </Space>
          </Form>
        )}

        {step === 2 && (
          <Form form={adminForm} layout="vertical" initialValues={{ admin_username: 'admin' }}>
            <Form.Item
              name="admin_username"
              label="管理员用户名"
              rules={[{ required: true, min: 2, max: 64, message: '2-64 个字符' }]}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="admin_password"
              label="管理员密码"
              rules={[{ required: true, min: 6, max: 128, message: '至少 6 位' }]}
            >
              <Input.Password placeholder="请输入密码" />
            </Form.Item>
            <Button type="primary" block loading={submitting} onClick={onFinish}>
              完成初始化
            </Button>
            <Button type="link" block onClick={() => setStep(1)}>
              上一步
            </Button>
          </Form>
        )}

        {step === 3 && (
          <div style={{ textAlign: 'center', padding: '24px 0' }}>
            <CheckCircleOutlined style={{ fontSize: 64, color: '#52c41a' }} />
            <Title level={4} style={{ marginTop: 16 }}>
              初始化完成
            </Title>
            <Text type="secondary">配置已保存。请重启服务后使用管理员账号登录。</Text>
          </div>
        )}
      </Card>
    </div>
  )
}

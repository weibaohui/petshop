import { useState, useEffect } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  message,
  Popconfirm,
  Typography,
  Descriptions,
} from 'antd';
import {
  PlusOutlined,
  CopyOutlined,
  DeleteOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  KeyOutlined,
  EyeOutlined,
  EyeInvisibleOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { listTokens, createToken, updateTokenStatus, deleteToken } from '../api/token';
import type { APIToken, APITokenCreateResponse } from '../types/api_token';

const { Title, Text } = Typography;
const { Option } = Select;
const { TextArea } = Input;

interface TokenFormValues {
  name: string;
  description?: string;
  expiresDays?: number;
}

export function TokenManagementPage() {
  const [tokens, setTokens] = useState<APIToken[]>([]);
  const [loading, setLoading] = useState(false);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isResultModalOpen, setIsResultModalOpen] = useState(false);
  const [createdToken, setCreatedToken] = useState<APITokenCreateResponse | null>(null);
  const [showToken, setShowToken] = useState(false);
  const [form] = Form.useForm<TokenFormValues>();
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  });

  const fetchTokens = async (page: number = 1, pageSize: number = 10) => {
    setLoading(true);
    try {
      const response = await listTokens(page, pageSize);
      setTokens(response.list);
      setPagination({
        current: page,
        pageSize: pageSize,
        total: response.total,
      });
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTokens();
  }, []);

  const handleCreate = async (values: TokenFormValues) => {
    try {
      const response = await createToken({
        name: values.name,
        description: values.description,
        expiresDays: values.expiresDays,
      });
      setCreatedToken(response);
      setIsModalOpen(false);
      setIsResultModalOpen(true);
      setShowToken(true);
      form.resetFields();
      fetchTokens(pagination.current, pagination.pageSize);
      message.success('Token 创建成功');
    } catch (error) {
      message.error(error instanceof Error ? error.message : '创建失败');
    }
  };

  const handleStatusChange = async (id: number, status: 'active' | 'disabled') => {
    try {
      await updateTokenStatus(id, { status });
      message.success(`Token 已${status === 'active' ? '启用' : '禁用'}`);
      fetchTokens(pagination.current, pagination.pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '操作失败');
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await deleteToken(id);
      message.success('Token 已删除');
      fetchTokens(pagination.current, pagination.pageSize);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除失败');
    }
  };

  const copyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      message.success('已复制到剪贴板');
    } catch {
      // Fallback for browsers that don't support clipboard API
      const textArea = document.createElement('textarea');
      textArea.value = text;
      textArea.style.position = 'fixed';
      textArea.style.left = '-9999px';
      document.body.appendChild(textArea);
      textArea.select();
      try {
        document.execCommand('copy');
        message.success('已复制到剪贴板');
      } catch {
        message.error('复制失败，请手动复制');
      }
      document.body.removeChild(textArea);
    }
  };

  const formatDate = (dateStr: string | null) => {
    if (!dateStr) return '-';
    return new Date(dateStr).toLocaleString('zh-CN');
  };

  const columns: ColumnsType<APIToken> = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => (
        <Space>
          <KeyOutlined />
          <Text strong>{text}</Text>
        </Space>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      render: (text: string) => text || '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'active' ? 'success' : 'default'}>
          {status === 'active' ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '最后使用',
      dataIndex: 'lastUsedAt',
      key: 'lastUsedAt',
      render: (date: string | null) => formatDate(date),
    },
    {
      title: '过期时间',
      dataIndex: 'expiresAt',
      key: 'expiresAt',
      render: (date: string | null) => date ? formatDate(date) : <Tag>永不过期</Tag>,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (date: string) => formatDate(date),
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_, record) => (
        <Space size="small">
          <Tooltip title={record.status === 'active' ? '禁用' : '启用'}>
            <Button
              type="text"
              icon={record.status === 'active' ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
              onClick={() => handleStatusChange(record.id, record.status === 'active' ? 'disabled' : 'active')}
            />
          </Tooltip>
          <Popconfirm
            title="确定要删除这个 Token 吗？"
            description="删除后将无法恢复，使用此 Token 的 API 调用将失败。"
            onConfirm={() => handleDelete(record.id)}
            okText="删除"
            okButtonProps={{ danger: true }}
            cancelText="取消"
          >
            <Tooltip title="删除">
              <Button type="text" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '24px' }}>
      <Card>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
          <div>
            <Title level={3} style={{ margin: 0 }}>API Token 管理</Title>
            <Text type="secondary">管理用于访问开放 API 的 Token</Text>
          </div>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setIsModalOpen(true)}
          >
            创建 Token
          </Button>
        </div>

        <Table
          columns={columns}
          dataSource={tokens}
          rowKey="id"
          loading={loading}
          pagination={{
            ...pagination,
            onChange: (page, pageSize) => fetchTokens(page, pageSize),
          }}
        />
      </Card>

      {/* Create Token Modal */}
      <Modal
        title="创建 API Token"
        open={isModalOpen}
        onOk={() => form.submit()}
        onCancel={() => {
          setIsModalOpen(false);
          form.resetFields();
        }}
        okText="创建"
        cancelText="取消"
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleCreate}
        >
          <Form.Item
            name="name"
            label="Token 名称"
            rules={[{ required: true, message: '请输入 Token 名称' }]}
          >
            <Input placeholder="例如：生产环境调用" maxLength={100} />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
          >
            <TextArea placeholder="描述这个 Token 的用途" rows={3} maxLength={500} showCount />
          </Form.Item>

          <Form.Item
            name="expiresDays"
            label="过期时间"
            initialValue={0}
          >
            <Select placeholder="选择过期时间">
              <Option value={0}>永不过期</Option>
              <Option value={7}>7 天</Option>
              <Option value={30}>30 天</Option>
              <Option value={90}>90 天</Option>
              <Option value={180}>180 天</Option>
              <Option value={365}>1 年</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* Token Result Modal */}
      <Modal
        title="Token 创建成功"
        open={isResultModalOpen}
        onOk={() => setIsResultModalOpen(false)}
        onCancel={() => setIsResultModalOpen(false)}
        okText="完成"
        closable={false}
        maskClosable={false}
      >
        {createdToken && (
          <div>
            <div style={{ marginBottom: 16 }}>
              <Text type="warning" strong>
                请立即复制并保存此 Token，它只会显示一次！
              </Text>
            </div>

            <Descriptions bordered column={1} size="small">
              <Descriptions.Item label="名称">{createdToken.name}</Descriptions.Item>
              <Descriptions.Item label="描述">{createdToken.description || '-'}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color="success">启用</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="过期时间">
                {createdToken.expiresAt ? formatDate(createdToken.expiresAt) : <Tag>永不过期</Tag>}
              </Descriptions.Item>
            </Descriptions>

            <div style={{ marginTop: 16 }}>
              <Text strong>Token:</Text>
              <Input.Group compact style={{ marginTop: 8 }}>
                <Input
                  style={{ width: 'calc(100% - 80px)' }}
                  type={showToken ? 'text' : 'password'}
                  value={createdToken.token}
                  readOnly
                />
                <Button
                  icon={showToken ? <EyeInvisibleOutlined /> : <EyeOutlined />}
                  onClick={() => setShowToken(!showToken)}
                />
                <Button
                  type="primary"
                  icon={<CopyOutlined />}
                  onClick={() => createdToken.token && copyToClipboard(createdToken.token)}
                >
                  复制
                </Button>
              </Input.Group>
            </div>

            <div style={{ marginTop: 16, padding: 12, background: '#f6ffed', border: '1px solid #b7eb8f', borderRadius: 4 }}>
              <Text type="secondary" style={{ fontSize: 12 }}>
                <strong>使用方式：</strong><br />
                在调用开放 API 时，在请求头中添加：<br />
                <code>X-API-Key: {createdToken.token}</code><br />
                或<br />
                <code>Authorization: Bearer {createdToken.token}</code>
              </Text>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}

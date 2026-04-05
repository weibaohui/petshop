import { useEffect, useState } from 'react';
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
  message,
  Popconfirm,
  Descriptions,
  Typography,
  Tooltip,
  Badge,
  Alert,
} from 'antd';
import {
  PlusOutlined,
  CopyOutlined,
  DeleteOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  KeyOutlined,
} from '@ant-design/icons';
import type { APIToken } from '../types/api_token';
import {
  listAPITokens,
  createAPIToken,
  updateAPITokenStatus,
  deleteAPIToken,
} from '../api/api_token';

const { Title, Text, Paragraph } = Typography;
const { Option } = Select;

export function APITokenManagePage() {
  const [tokens, setTokens] = useState<APIToken[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [loading, setLoading] = useState(false);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isTokenModalOpen, setIsTokenModalOpen] = useState(false);
  const [newToken, setNewToken] = useState<APIToken | null>(null);
  const [form] = Form.useForm();

  const fetchTokens = async () => {
    setLoading(true);
    try {
      const res = await listAPITokens(page, pageSize);
      setTokens(res.list);
      setTotal(res.total);
    } catch (error) {
      message.error(error instanceof Error ? error.message : '获取列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTokens();
  }, [page, pageSize]);

  const handleCreate = async (values: {
    name: string;
    expiresDays?: number;
    permissions: string[];
  }) => {
    try {
      const res = await createAPIToken({
        name: values.name,
        expiresDays: values.expiresDays || 0,
        permissions: values.permissions?.join(',') || 'read',
      });
      setNewToken(res);
      setIsCreateModalOpen(false);
      setIsTokenModalOpen(true);
      form.resetFields();
      fetchTokens();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '创建失败');
    }
  };

  const handleCopyToken = (token: string) => {
    navigator.clipboard.writeText(token);
    message.success('Token 已复制到剪贴板');
  };

  const handleToggleStatus = async (token: APIToken) => {
    try {
      const newStatus = token.status === 'active' ? 'disabled' : 'active';
      await updateAPITokenStatus(token.id, { status: newStatus });
      message.success('状态更新成功');
      fetchTokens();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '更新失败');
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await deleteAPIToken(id);
      message.success('删除成功');
      fetchTokens();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除失败');
    }
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString('zh-CN');
  };

  const getStatusBadge = (status: string) => {
    return status === 'active' ? (
      <Badge status="success" text="启用" />
    ) : (
      <Badge status="error" text="禁用" />
    );
  };

  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => getStatusBadge(status),
    },
    {
      title: '权限',
      dataIndex: 'permissions',
      key: 'permissions',
      width: 150,
      render: (permissions: string) => {
        const perms = permissions?.split(',') || ['read'];
        return (
          <Space size="small">
            {perms.map((p) => (
              <Tag key={p} color={p === 'admin' ? 'red' : p === 'write' ? 'blue' : 'green'}>
                {p === 'read' ? '读取' : p === 'write' ? '写入' : p === 'admin' ? '管理' : p}
              </Tag>
            ))}
          </Space>
        );
      },
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (date: string) => formatDate(date),
    },
    {
      title: '过期时间',
      dataIndex: 'expiresAt',
      key: 'expiresAt',
      width: 180,
      render: (date?: string) => (date ? formatDate(date) : <Text type="secondary">永不过期</Text>),
    },
    {
      title: '最后使用',
      dataIndex: 'lastUsedAt',
      key: 'lastUsedAt',
      width: 180,
      render: (date?: string) => (date ? formatDate(date) : <Text type="secondary">从未使用</Text>),
    },
    {
      title: '操作',
      key: 'action',
      width: 150,
      fixed: 'right' as const,
      render: (_: unknown, record: APIToken) => (
        <Space size="small">
          <Tooltip title={record.status === 'active' ? '禁用' : '启用'}>
            <Button
              type="text"
              icon={record.status === 'active' ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
              onClick={() => handleToggleStatus(record)}
            />
          </Tooltip>
          <Popconfirm
            title="确认删除"
            description="删除后无法恢复，确定要删除吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '24px', maxWidth: '1400px', margin: '0 auto' }}>
      <Card>
        <Space direction="vertical" style={{ width: '100%' }} size="large">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div>
              <Title level={3} style={{ margin: 0 }}>
                <KeyOutlined style={{ marginRight: 8 }} />
                API Token 管理
              </Title>
              <Paragraph type="secondary">管理开放 API 的访问令牌，用于第三方系统集成</Paragraph>
            </div>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setIsCreateModalOpen(true)}
            >
              创建 Token
            </Button>
          </div>

          <Table
            dataSource={tokens}
            columns={columns}
            rowKey="id"
            loading={loading}
            pagination={{
              current: page,
              pageSize: pageSize,
              total: total,
              showSizeChanger: true,
              showTotal: (total) => `共 ${total} 条`,
              onChange: (p, s) => {
                setPage(p);
                if (s) setPageSize(s);
              },
            }}
            scroll={{ x: 1000 }}
          />
        </Space>
      </Card>

      {/* 创建 Token 模态框 */}
      <Modal
        title="创建 API Token"
        open={isCreateModalOpen}
        onCancel={() => setIsCreateModalOpen(false)}
        footer={null}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleCreate}>
          <Form.Item
            name="name"
            label="Token 名称"
            rules={[
              { required: true, message: '请输入 Token 名称' },
              { max: 100, message: '名称不能超过 100 个字符' },
            ]}
          >
            <Input placeholder="例如：第三方系统集成" />
          </Form.Item>

          <Form.Item
            name="permissions"
            label="权限"
            initialValue={['read']}
            rules={[{ required: true, message: '请选择权限' }]}
          >
            <Select mode="multiple" placeholder="选择权限">
              <Option value="read">读取 (read)</Option>
              <Option value="write">写入 (write)</Option>
              <Option value="admin">管理 (admin)</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="expiresDays"
            label="有效期（天）"
            tooltip="留空表示永不过期"
          >
            <Select placeholder="选择有效期" allowClear>
              <Option value={0}>永不过期</Option>
              <Option value={7}>7 天</Option>
              <Option value={30}>30 天</Option>
              <Option value={90}>90 天</Option>
              <Option value={365}>1 年</Option>
            </Select>
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setIsCreateModalOpen(false)}>取消</Button>
              <Button type="primary" htmlType="submit">
                创建
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 显示新生成的 Token */}
      <Modal
        title="API Token 创建成功"
        open={isTokenModalOpen}
        onCancel={() => setIsTokenModalOpen(false)}
        footer={[
          <Button key="ok" type="primary" onClick={() => setIsTokenModalOpen(false)}>
            我已保存
          </Button>,
        ]}
        width={600}
      >
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <Alert
            message="请立即复制并安全保存此 Token"
            description="此 Token 仅显示一次，关闭后将无法再次查看。请将其保存在安全的地方。"
            type="warning"
            showIcon
          />

          <Card>
            <Descriptions column={1} size="small">
              <Descriptions.Item label="名称">{newToken?.name}</Descriptions.Item>
              <Descriptions.Item label="权限">
                {newToken?.permissions?.split(',').map((p) => (
                  <Tag key={p} color={p === 'admin' ? 'red' : p === 'write' ? 'blue' : 'green'}>
                    {p}
                  </Tag>
                ))}
              </Descriptions.Item>
              <Descriptions.Item label="过期时间">
                {newToken?.expiresAt
                  ? formatDate(newToken.expiresAt)
                  : '永不过期'}
              </Descriptions.Item>
            </Descriptions>

            <div style={{ marginTop: 16 }}>
              <Text strong>Token：</Text>
              <Input.Group compact>
                <Input
                  style={{ width: 'calc(100% - 100px)' }}
                  value={newToken?.token}
                  readOnly
                  type="password"
                  addonBefore={<KeyOutlined />}
                />
                <Button
                  type="primary"
                  icon={<CopyOutlined />}
                  onClick={() => newToken?.token && handleCopyToken(newToken.token)}
                >
                  复制
                </Button>
              </Input.Group>
            </div>

            <div style={{ marginTop: 16 }}>
              <Text strong>使用示例：</Text>
              <pre
                style={{
                  background: '#f5f5f5',
                  padding: 12,
                  borderRadius: 4,
                  overflow: 'auto',
                }}
              >
{`curl -X GET \\
  'https://api.example.com/api/open/pets' \\
  -H 'Authorization: Bearer ${newToken?.token}'`}
              </pre>
            </div>
          </Card>
        </Space>
      </Modal>
    </div>
  );
}

import { useEffect, useState } from 'react';
import {
  Layout,
  Input,
  Select,
  Row,
  Col,
  Pagination,
  Typography,
  Space,
  Spin,
  Empty,
  Card,
  Slider,
  Alert,
} from 'antd';
import { SearchOutlined } from '@ant-design/icons';
import { PetCard } from '../components/PetCard';
import { getPets, getCategories } from '../api/pet';
import type { Pet, Category, PetFilter } from '../types/pet';

const { Header, Content, Footer } = Layout;
const { Title, Text } = Typography;
const { Search } = Input;

const PAGE_SIZE = 8;

export function HomePage() {
  const [pets, setPets] = useState<Pet[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [total, setTotal] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const [filter, setFilter] = useState<PetFilter>({
    page: 1,
    pageSize: PAGE_SIZE,
  });
  // 临时存储价格滑块值，避免频繁触发请求
  const [tempPriceRange, setTempPriceRange] = useState<[number, number]>([0, 10000]);

  useEffect(() => {
    fetchCategories();
    fetchPets();
  }, []);

  useEffect(() => {
    fetchPets();
  }, [filter]);

  const fetchCategories = async () => {
    try {
      const data = await getCategories();
      setCategories(data);
    } catch (error) {
      console.error('Failed to fetch categories:', error);
    }
  };

  const fetchPets = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await getPets(filter);
      setPets(response.data);
      setTotal(response.page.total);
      setCurrentPage(response.page.page);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : '获取宠物列表失败，请稍后重试';
      setError(errorMessage);
      console.error('Failed to fetch pets:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = (value: string) => {
    setFilter((prev) => ({ ...prev, search: value, page: 1 }));
  };

  const handleCategoryChange = (value: string) => {
    setFilter((prev) => ({ ...prev, type: value, page: 1 }));
  };

  // 滑块拖动过程中只更新临时状态，不触发请求
  const handlePriceChange = (value: number[]) => {
    setTempPriceRange([value[0], value[1]]);
  };

  // 滑块拖动结束时才更新 filter 触发请求
  const handlePriceChangeComplete = (value: number[]) => {
    setFilter((prev) => ({
      ...prev,
      minPrice: value[0],
      maxPrice: value[1],
      page: 1,
    }));
  };

  const handlePageChange = (page: number) => {
    setFilter((prev) => ({ ...prev, page }));
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  return (
    <Layout style={{ minHeight: '100vh', background: '#f5f5f5' }}>
      {/* Header */}
      <Header
        style={{
          background: '#fff',
          boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
          position: 'sticky',
          top: 0,
          zIndex: 100,
          padding: '0 24px',
        }}
      >
        <div
          style={{
            maxWidth: 1400,
            margin: '0 auto',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            height: '100%',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <div
              style={{
                width: 40,
                height: 40,
                borderRadius: '50%',
                background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <span style={{ color: '#fff', fontSize: 20 }}>🐾</span>
            </div>
            <Title level={3} style={{ margin: 0, color: '#1890ff' }}>
              Pet Shop
            </Title>
          </div>
          <Search
            placeholder="搜索宠物名称或品种..."
            allowClear
            enterButton={<SearchOutlined />}
            onSearch={handleSearch}
            style={{ width: 300 }}
          />
        </div>
      </Header>

      {/* Hero Banner */}
      <div
        style={{
          background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
          padding: '60px 24px',
          textAlign: 'center',
        }}
      >
        <Title level={1} style={{ color: '#fff', marginBottom: 16 }}>
          找到你的完美伙伴
        </Title>
        <Text style={{ color: 'rgba(255,255,255,0.9)', fontSize: 18 }}>
          精选优质宠物，陪伴你的每一天
        </Text>
      </div>

      {/* Main Content */}
      <Content style={{ maxWidth: 1400, margin: '0 auto', padding: '24px', width: '100%' }}>
        {/* Filter Bar */}
        <Card style={{ marginBottom: 24 }}>
          <Row gutter={[24, 16]} align="middle">
            <Col xs={24} sm={12} md={6}>
              <Space direction="vertical" style={{ width: '100%' }}>
                <Text strong>宠物分类</Text>
                <Select
                  placeholder="选择分类"
                  allowClear
                  style={{ width: '100%' }}
                  onChange={handleCategoryChange}
                  options={categories.map((c) => ({ label: c.name, value: c.name }))}
                />
              </Space>
            </Col>
            <Col xs={24} sm={12} md={10}>
              <Space direction="vertical" style={{ width: '100%' }}>
                <Text strong>价格区间: ¥{tempPriceRange[0]} - ¥{tempPriceRange[1]}</Text>
                <Slider
                  range
                  min={0}
                  max={10000}
                  step={100}
                  value={tempPriceRange}
                  onChange={handlePriceChange}
                  onChangeComplete={handlePriceChangeComplete}
                  tooltip={{ formatter: (value) => `¥${value}` }}
                />
              </Space>
            </Col>
          </Row>
        </Card>

        {/* Error Alert */}
        {error && (
          <Alert
            message="加载失败"
            description={error}
            type="error"
            showIcon
            closable
            onClose={() => setError(null)}
            style={{ marginBottom: 24 }}
          />
        )}

        {/* Pet Grid */}
        <Spin spinning={loading} size="large">
          {pets.length > 0 ? (
            <>
              <Row gutter={[24, 24]}>
                {pets.map((pet) => (
                  <Col key={pet.id} xs={24} sm={12} md={8} lg={6}>
                    <PetCard pet={pet} />
                  </Col>
                ))}
              </Row>
              <div style={{ textAlign: 'center', marginTop: 48 }}>
                <Pagination
                  current={currentPage}
                  pageSize={PAGE_SIZE}
                  total={total}
                  onChange={handlePageChange}
                  showSizeChanger={false}
                  showQuickJumper
                  showTotal={(total) => `共 ${total} 只宠物`}
                />
              </div>
            </>
          ) : (
            <Empty
              description="暂无符合条件的宠物"
              style={{ padding: 60 }}
            />
          )}
        </Spin>
      </Content>

      {/* Footer */}
      <Footer style={{ background: '#001529', color: '#fff', padding: '48px 24px' }}>
        <div style={{ maxWidth: 1400, margin: '0 auto' }}>
          <Row gutter={[48, 24]}>
            <Col xs={24} md={8}>
              <Title level={4} style={{ color: '#fff' }}>
                Pet Shop
              </Title>
              <Text style={{ color: 'rgba(255,255,255,0.65)' }}>
                专业的宠物销售平台，为您提供健康优质的宠物伙伴。
              </Text>
            </Col>
            <Col xs={24} md={8}>
              <Title level={4} style={{ color: '#fff' }}>
                联系我们
              </Title>
              <Text style={{ color: 'rgba(255,255,255,0.65)' }}>
                电话: 400-888-8888
                <br />
                邮箱: contact@petshop.com
                <br />
                地址: 北京市朝阳区宠物大街88号
              </Text>
            </Col>
            <Col xs={24} md={8}>
              <Title level={4} style={{ color: '#fff' }}>
                营业时间
              </Title>
              <Text style={{ color: 'rgba(255,255,255,0.65)' }}>
                周一至周五: 9:00 - 21:00
                <br />
                周末及节假日: 10:00 - 22:00
              </Text>
            </Col>
          </Row>
          <div
            style={{
              borderTop: '1px solid rgba(255,255,255,0.1)',
              marginTop: 48,
              paddingTop: 24,
              textAlign: 'center',
            }}
          >
            <Text style={{ color: 'rgba(255,255,255,0.45)' }}>
              © 2024 Pet Shop. All rights reserved.
            </Text>
          </div>
        </div>
      </Footer>
    </Layout>
  );
}

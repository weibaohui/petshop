import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Layout,
  Card,
  Row,
  Col,
  Image,
  Typography,
  Tag,
  Descriptions,
  Button,
  Spin,
  Empty,
  Carousel,
  Timeline,
  message,
} from 'antd';
import { LeftOutlined, CalendarOutlined, MedicineBoxOutlined } from '@ant-design/icons';
import { getPetById } from '../api/pet';
import type { Pet, VaccinationRecord } from '../types/pet';
import { StatusMap } from '../types/pet';

const { Header, Content, Footer } = Layout;
const { Title, Text, Paragraph } = Typography;

export function PetDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [pet, setPet] = useState<Pet | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (id) {
      fetchPet(parseInt(id, 10));
    }
  }, [id]);

  const fetchPet = async (petId: number) => {
    setLoading(true);
    try {
      const data = await getPetById(petId);
      setPet(data);
    } catch (error) {
      console.error('Failed to fetch pet:', error);
      message.error('加载宠物信息失败');
    } finally {
      setLoading(false);
    }
  };

  const handleBooking = () => {
    message.success('预约申请已提交，我们会尽快联系您！');
  };

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <Spin size="large" tip="加载中..." />
      </div>
    );
  }

  if (!pet) {
    return (
      <Layout style={{ minHeight: '100vh' }}>
        <Content style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
          <Empty description="宠物不存在或已被删除" />
        </Content>
      </Layout>
    );
  }

  const status = StatusMap[pet.status] || { text: '未知', color: 'default' };

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
            height: '100%',
          }}
        >
          <Button icon={<LeftOutlined />} onClick={() => navigate(-1)}>
            返回
          </Button>
          <Title level={4} style={{ margin: 0, marginLeft: 16 }}>
            宠物详情
          </Title>
        </div>
      </Header>

      {/* Main Content */}
      <Content style={{ maxWidth: 1400, margin: '0 auto', padding: '24px', width: '100%' }}>
        <Row gutter={[24, 24]}>
          {/* Image Carousel */}
          <Col xs={24} md={12}>
            <Card>
              {pet.photoUrls.length > 0 ? (
                <Carousel autoplay effect="fade">
                  {pet.photoUrls.map((url, index) => (
                    <div key={index}>
                      <Image
                        src={url}
                        alt={`${pet.name} - ${index + 1}`}
                        style={{ width: '100%', height: 500, objectFit: 'cover' }}
                        preview
                      />
                    </div>
                  ))}
                </Carousel>
              ) : (
                <div
                  style={{
                    height: 500,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: '#f0f0f0',
                  }}
                >
                  <Text type="secondary">暂无图片</Text>
                </div>
              )}
            </Card>
          </Col>

          {/* Pet Info */}
          <Col xs={24} md={12}>
            <Card
              title={
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Title level={3} style={{ margin: 0 }}>
                    {pet.name}
                  </Title>
                  <Tag color={status.color} style={{ fontSize: 14, padding: '4px 12px' }}>
                    {status.text}
                  </Tag>
                </div>
              }
            >
              <div style={{ marginBottom: 24 }}>
                <Text style={{ fontSize: 32, color: '#f5222d', fontWeight: 'bold' }}>
                  ¥{pet.price.toLocaleString()}
                </Text>
              </div>

              <Descriptions column={2} bordered style={{ marginBottom: 24 }}>
                <Descriptions.Item label="品种">{pet.breed}</Descriptions.Item>
                <Descriptions.Item label="类型">{pet.type}</Descriptions.Item>
                <Descriptions.Item label="年龄">{pet.ageDisplay}</Descriptions.Item>
                <Descriptions.Item label="发布时间">{pet.createdAt}</Descriptions.Item>
              </Descriptions>

              <div style={{ marginBottom: 24 }}>
                <Title level={5}>健康状态</Title>
                <Paragraph>{pet.healthStatus}</Paragraph>
              </div>

              <div style={{ marginBottom: 24 }}>
                <Title level={5}>详细描述</Title>
                <Paragraph>{pet.description}</Paragraph>
              </div>

              <Button
                type="primary"
                size="large"
                block
                onClick={handleBooking}
                disabled={pet.status !== 'available'}
                style={{ height: 50, fontSize: 18 }}
              >
                {pet.status === 'available' ? '预约看宠' : '暂不可预约'}
              </Button>
            </Card>

            {/* Vaccination Records */}
            {pet.vaccinationRecords.length > 0 && (
              <Card title="疫苗接种记录" style={{ marginTop: 24 }}>
                <Timeline>
                  {pet.vaccinationRecords.map((record: VaccinationRecord, index: number) => (
                    <Timeline.Item
                      key={index}
                      dot={<MedicineBoxOutlined />}
                      color={record.completed ? 'green' : 'gray'}
                    >
                      <div>
                        <Text strong>{record.name}</Text>
                        <br />
                        <Text type="secondary">
                          <CalendarOutlined /> {record.date}
                        </Text>
                        <br />
                        <Tag color={record.completed ? 'success' : 'default'}>
                          {record.completed ? '已完成' : '待接种'}
                        </Tag>
                      </div>
                    </Timeline.Item>
                  ))}
                </Timeline>
              </Card>
            )}
          </Col>
        </Row>
      </Content>

      {/* Footer */}
      <Footer style={{ background: '#001529', color: '#fff', padding: '48px 24px' }}>
        <div style={{ maxWidth: 1400, margin: '0 auto', textAlign: 'center' }}>
          <Text style={{ color: 'rgba(255,255,255,0.45)' }}>
            © 2024 Pet Shop. All rights reserved.
          </Text>
        </div>
      </Footer>
    </Layout>
  );
}

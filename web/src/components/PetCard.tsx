import { Card, Tag, Typography, Button } from 'antd';
import { EyeOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import type { Pet } from '../types/pet';
import { StatusMap } from '../types/pet';

const { Meta } = Card;
const { Text } = Typography;

interface PetCardProps {
  pet: Pet;
}

export function PetCard({ pet }: PetCardProps) {
  const navigate = useNavigate();
  const status = StatusMap[pet.status] || { text: '未知', color: 'default' };

  return (
    <Card
      hoverable
      cover={
        <div style={{ height: 200, overflow: 'hidden', position: 'relative' }}>
          <img
            alt={pet.name}
            src={pet.photoUrls[0] || 'https://via.placeholder.com/400x400?text=No+Image'}
            style={{
              width: '100%',
              height: '100%',
              objectFit: 'cover',
            }}
            loading="lazy"
          />
          <Tag
            color={status.color}
            style={{
              position: 'absolute',
              top: 8,
              right: 8,
              fontSize: 12,
              padding: '2px 8px',
            }}
          >
            {status.text}
          </Tag>
        </div>
      }
      actions={[
        <Button
          type="primary"
          icon={<EyeOutlined />}
          onClick={() => navigate(`/pet/${pet.id}`)}
          block
          style={{ margin: '0 8px', width: 'calc(100% - 16px)' }}
        >
          查看详情
        </Button>,
      ]}
    >
      <Meta
        title={
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span style={{ fontSize: 18, fontWeight: 600 }}>{pet.name}</span>
            <span style={{ fontSize: 16, color: '#f5222d', fontWeight: 600 }}>
              ¥{pet.price.toLocaleString()}
            </span>
          </div>
        }
        description={
          <div style={{ marginTop: 8 }}>
            <div style={{ marginBottom: 4 }}>
              <Text type="secondary">品种：</Text>
              <Text>{pet.breed}</Text>
            </div>
            <div>
              <Text type="secondary">年龄：</Text>
              <Text>{pet.ageDisplay}</Text>
            </div>
          </div>
        }
      />
    </Card>
  );
}

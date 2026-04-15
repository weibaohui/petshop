import { Card, Tag, Typography, Button, message } from 'antd';
import { EyeOutlined, StarOutlined, StarFilled } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useState, useEffect } from 'react';
import type { Pet } from '../types/pet';
import { StatusMap } from '../types/pet';
import { voteForPet, getVoteStatus } from '../api/vote';

const { Meta } = Card;
const { Text } = Typography;

interface PetCardProps {
  pet: Pet;
}

const CURRENT_USER_ID = 1;

export function PetCard({ pet }: PetCardProps) {
  const navigate = useNavigate();
  const status = StatusMap[pet.status] || { text: '未知', color: 'default' };
  const [voteCount, setVoteCount] = useState(pet.voteCount || 0);
  const [hasVoted, setHasVoted] = useState(false);
  const [voting, setVoting] = useState(false);

  useEffect(() => {
    checkVoteStatus();
  }, [pet.id]);

  const checkVoteStatus = async () => {
    try {
      const status = await getVoteStatus(pet.id, CURRENT_USER_ID);
      setVoteCount(status.voteCount);
      setHasVoted(status.hasVoted);
    } catch (error) {
      console.error('Failed to check vote status:', error);
    }
  };

  const handleVote = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (hasVoted) {
      message.warning('您已经为这只宠物投过票了');
      return;
    }
    setVoting(true);
    try {
      const result = await voteForPet(pet.id, CURRENT_USER_ID);
      setVoteCount(result.voteCount);
      setHasVoted(result.hasVoted);
      message.success('投票成功！');
    } catch (error) {
      message.error(error instanceof Error ? error.message : '投票失败');
    } finally {
      setVoting(false);
    }
  };

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
          type={hasVoted ? 'default' : 'primary'}
          icon={hasVoted ? <StarFilled /> : <StarOutlined />}
          onClick={handleVote}
          loading={voting}
          block
          style={{ margin: '0 8px', width: 'calc(100% - 16px)', backgroundColor: hasVoted ? '#faad14' : undefined }}
        >
          {hasVoted ? `已投票 (${voteCount})` : `投票 (${voteCount})`}
        </Button>,
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

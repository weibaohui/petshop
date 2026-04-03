# Pet Shop

宠物商店展示平台 - 一个现代化的宠物展示与销售系统。

## 项目概述

这是一个全栈宠物商店项目，包含：

- **后端**: Go + 标准库实现的 REST API
- **前端**: React + TypeScript + Ant Design

## 功能特性

### 核心功能
- 宠物展示首页，无需登录即可浏览
- 宠物分类筛选（狗狗、猫咪、鸟类、其他）
- 价格区间筛选
- 关键字搜索（支持名称、品种）
- 宠物详情页面（大图轮播、健康信息、疫苗记录）
- 预约看宠功能

### 技术特点
- 响应式设计，支持桌面端和移动端
- 图片懒加载优化
- 缓存机制提升性能
- RESTful API 设计

## 项目结构

```text
.
├── cmd/server/          # 后端入口
├── internal/            # 后端内部包
│   ├── handlers/        # HTTP 处理器
│   ├── models/          # 数据模型
│   ├── middleware/      # 中间件
│   └── ...
└── web/                 # 前端项目
    ├── src/
    │   ├── api/         # API 调用
    │   ├── components/  # 组件
    │   ├── pages/       # 页面
    │   └── types/       # TypeScript 类型
    └── ...
```

## 快速开始

### 后端启动

```bash
# 克隆项目
git clone git@github.com:weibaohui/petshop.git
cd petshop

# 初始化
make setup

# 启动服务器
go run cmd/server/main.go
```

后端服务将在 http://localhost:8080 启动

### 前端启动

```bash
cd web
npm install
npm run dev
```

前端开发服务器将在 http://localhost:3000 启动

## API 文档

### 宠物相关接口

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/v1/pets` | 获取宠物列表（支持筛选） |
| GET | `/api/v1/pets/:id` | 获取单个宠物详情 |
| GET | `/api/v1/categories` | 获取宠物分类 |
| GET | `/api/pets` | 获取宠物列表（旧接口） |
| GET | `/api/pet?id=1` | 获取单个宠物 |

### 查询参数

获取宠物列表支持以下参数：

- `type` - 宠物类型（狗狗/猫咪/鸟类/其他）
- `status` - 状态（available/pending/sold）
- `search` - 搜索关键字
- `minPrice` - 最低价格
- `maxPrice` - 最高价格
- `page` - 页码
- `pageSize` - 每页数量

## 数据模型

### Pet
```typescript
{
  id: number;
  name: string;           // 宠物名称
  type: string;           // 类型
  breed: string;          // 品种
  photoUrls: string[];    // 照片URL
  status: string;         // 状态
  age: number;            // 年龄（月）
  ageDisplay: string;     // 年龄显示
  price: number;          // 价格
  description: string;    // 描述
  healthStatus: string;   // 健康状况
  vaccinationRecords: VaccinationRecord[];
  createdAt: string;
}
```

## 开发

### 运行测试

```bash
# 后端测试
go test ./...

# 前端代码检查
cd web
npm run lint
```

### 构建

```bash
# 构建前端
cd web
npm run build

# 构建后端
go build -o petshop cmd/server/main.go
```

## 许可证

MIT

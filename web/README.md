# Pet Shop 前端

宠物商店展示平台的前端项目，使用 React + TypeScript + Ant Design 构建。

## 技术栈

- **框架**: React 19
- **语言**: TypeScript
- **构建工具**: Vite
- **UI 组件库**: Ant Design 6.x
- **路由**: React Router 7

## 功能特性

- 宠物展示首页，支持分类筛选和价格区间筛选
- 宠物详情页面，展示大图轮播、健康信息、疫苗记录
- 响应式设计，支持桌面端和移动端
- 预约看宠功能

## 开发

### 安装依赖

```bash
npm install
```

### 启动开发服务器

```bash
npm run dev
```

开发服务器将在 http://localhost:3000 启动

### 构建生产版本

```bash
npm run build
```

### 代码检查

```bash
npm run lint
```

## 项目结构

```text
src/
├── api/          # API 调用封装
├── components/   # 通用组件
├── pages/        # 页面组件
├── types/        # TypeScript 类型定义
├── App.tsx       # 应用入口
└── main.tsx      # 渲染入口
```

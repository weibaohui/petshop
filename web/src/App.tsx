import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { HomePage } from './pages/HomePage';
import { PetDetailPage } from './pages/PetDetailPage';
import { TokenManagementPage } from './pages/TokenManagementPage';

function App() {
  return (
    <ConfigProvider locale={zhCN}>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/pet/:id" element={<PetDetailPage />} />
          <Route path="/admin/tokens" element={<TokenManagementPage />} />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  );
}

export default App;

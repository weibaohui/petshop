import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { HomePage } from './pages/HomePage';
import { PetDetailPage } from './pages/PetDetailPage';

function App() {
  return (
    <ConfigProvider locale={zhCN}>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/pet/:id" element={<PetDetailPage />} />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  );
}

export default App;

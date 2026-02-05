import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Layout } from './components/Layout';
import { ToastContainer } from './components/Toast';
import { SlavesPage } from './pages/SlavesPage';
import { InboundsPage } from './pages/InboundsPage';
import { XrayConfigPage } from './pages/XrayConfigPage';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
      staleTime: 30000,
    },
  },
});

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter basename="/panel">
        <Layout>
          <Routes>
            <Route path="/" element={<Navigate to="/slaves" replace />} />
            <Route path="/slaves" element={<SlavesPage />} />
            <Route path="/inbounds" element={<InboundsPage />} />
            <Route path="/xray/:slaveId" element={<XrayConfigPage />} />
            <Route path="*" element={<Navigate to="/slaves" replace />} />
          </Routes>
        </Layout>
        <ToastContainer />
      </BrowserRouter>
    </QueryClientProvider>
  );
}

export default App;

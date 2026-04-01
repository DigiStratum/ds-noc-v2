import { Routes, Route, Navigate } from 'react-router-dom';
import { Layout } from './app/Layout';
import { HomePage, NocDashboardPage, AutomationPage } from './app/pages';

export function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route path="/dashboard" element={<NocDashboardPage />} />
        <Route path="/automation" element={<AutomationPage />} />
        <Route path="/about" element={<HomePage />} />
        {/* Add your app routes here */}
      </Routes>
    </Layout>
  );
}

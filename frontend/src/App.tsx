import { Routes, Route } from 'react-router-dom';
import { Layout } from './app/Layout';
import { HomePage } from './app/pages';

export function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<HomePage />} />
        {/* Add your app routes here */}
      </Routes>
    </Layout>
  );
}

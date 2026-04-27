import { Routes, Route } from 'react-router-dom';
import { Layout } from './components/Layout';
import { Dashboard } from './pages/Dashboard';
import { MemoryList } from './pages/MemoryList';
import { MemoryDetail } from './pages/MemoryDetail';
import { PinnedPreview } from './pages/PinnedPreview';
import { Search } from './pages/Search';
import { Settings } from './pages/Settings';

export function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/memories" element={<MemoryList />} />
        <Route path="/memories/:id" element={<MemoryDetail />} />
        <Route path="/pinned" element={<PinnedPreview />} />
        <Route path="/search" element={<Search />} />
        <Route path="/settings" element={<Settings />} />
      </Routes>
    </Layout>
  );
}

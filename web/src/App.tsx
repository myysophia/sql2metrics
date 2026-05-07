import { Routes, Route, Navigate } from 'react-router-dom'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import DataSources from './pages/DataSources'
import Metrics from './pages/Metrics'
import Alerts from './pages/Alerts'
import AlertDetail from './pages/AlertDetail'
import Settings from './pages/Settings'
import NotificationSettings from './pages/NotificationSettings'
import RouteManagement from './pages/RouteManagement'
import AIChat from './pages/AIChat'

function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/datasources" element={<DataSources />} />
        <Route path="/metrics" element={<Metrics />} />
        <Route path="/alerts" element={<Alerts />} />
        <Route path="/alerts/:id" element={<AlertDetail />} />
        <Route path="/settings" element={<Settings />} />
        <Route path="/notifications" element={<NotificationSettings />} />
        <Route path="/routes" element={<RouteManagement />} />
        <Route path="/ai-chat" element={<AIChat />} />
      </Routes>
    </Layout>
  )
}

export default App

import { Routes, Route, Navigate } from 'react-router-dom'
import { Box } from '@mui/material'
import { useEffect } from 'react'

import { MainLayout } from '@/components/Layout/MainLayout'
import { Dashboard } from '@/pages/Dashboard'
import { Experiments } from '@/pages/Experiments'
import { ExperimentDetails } from '@/pages/ExperimentDetails'
import { PipelineBuilder } from '@/pages/PipelineBuilder'
import { Analysis } from '@/pages/Analysis'
import { Settings } from '@/pages/Settings'
import { Login } from '@/pages/Login'
import { PrivateRoute } from '@/components/Auth/PrivateRoute'
import { useAuth } from '@/hooks/useAuth'
import { useWebSocket } from '@/hooks/useWebSocket'

function App() {
  const { isAuthenticated, checkAuth } = useAuth()
  const { connect, disconnect } = useWebSocket()

  useEffect(() => {
    checkAuth()
  }, [checkAuth])

  useEffect(() => {
    if (isAuthenticated) {
      connect()
      return () => {
        disconnect()
      }
    }
  }, [isAuthenticated, connect, disconnect])

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      <Routes>
        <Route path="/login" element={<Login />} />
        
        <Route path="/" element={
          <PrivateRoute>
            <MainLayout />
          </PrivateRoute>
        }>
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<Dashboard />} />
          <Route path="experiments" element={<Experiments />} />
          <Route path="experiments/:id" element={<ExperimentDetails />} />
          <Route path="experiments/:id/analysis" element={<Analysis />} />
          <Route path="pipeline-builder" element={<PipelineBuilder />} />
          <Route path="settings" element={<Settings />} />
        </Route>
        
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Box>
  )
}

export default App
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Layout from './components/Layout'
import Dashboard from './components/Dashboard'
import MotorControl from './components/MotorControl'
import YamlGenerator from './components/YamlGenerator'

import EdsViewer from './components/EdsViewer'

function App() {
  return (
    <BrowserRouter>
      <Layout>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/motor" element={<MotorControl />} />
          <Route path="/yaml-gen" element={<YamlGenerator />} />
          <Route path="/eds" element={<EdsViewer />} />
        </Routes>
      </Layout>
    </BrowserRouter>
  )
}

export default App

import '@graphiql/react/setup-workers/vite'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import 'graphiql/style.css'
import './index.css'
import App from './App'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)

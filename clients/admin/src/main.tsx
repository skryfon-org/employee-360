import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import { configureApiClient } from '@employee360/api-client'
import './index.css'
import App from './App.tsx'

// Resolve the API base URL from this app's own environment and hand it to
// the framework-independent shared client once, at startup.
configureApiClient({ baseURL: import.meta.env.VITE_API_BASE_URL })

const queryClient = new QueryClient()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
)

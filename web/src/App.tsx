import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { GraphiQL } from 'graphiql'
import { createGraphiQLFetcher } from '@graphiql/toolkit'
import { MetricsPlugin } from './MetricsPlugin'
import AdminApp from './admin/AdminApp'

const fetcher = createGraphiQLFetcher({
  url: '/graphql',
  subscriptionUrl: window.location.origin.replace(/^http/, 'ws') + '/graphql',
})

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route
          path="/admin/*"
          element={<AdminApp />}
        />
        <Route
          path="/*"
          element={
            <GraphiQL
              fetcher={fetcher}
              plugins={[MetricsPlugin]}
              defaultEditorToolsVisibility
              isHeadersEditorEnabled
            />
          }
        />
      </Routes>
    </BrowserRouter>
  )
}

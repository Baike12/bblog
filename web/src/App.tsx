import { BrowserRouter, Route, Routes } from 'react-router-dom'

import { Layout } from './components/Layout'
import { bootstrap } from './lib/api'
import { Archive } from './routes/Archive'
import { Home } from './routes/Home'
import { NotFound } from './routes/NotFound'
import { Search } from './routes/Search'
import { TagDetail } from './routes/TagDetail'
import { TagIndex } from './routes/TagIndex'

export function App() {
  return (
    <BrowserRouter basename={bootstrap.site.base_path || '/'}>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<Home />} />
          <Route path="tags" element={<TagIndex />} />
          <Route path="tags/:tag" element={<TagDetail />} />
          <Route path="archive" element={<Archive />} />
          <Route path="search" element={<Search />} />
          <Route path="*" element={<NotFound />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

import { Navigate, Route, Routes } from 'react-router-dom'
import { Layout } from './components/Layout'
import { OverviewPage } from './pages/OverviewPage'
import { LecturersPage } from './pages/LecturersPage'
import { LocationsPage } from './pages/LocationsPage'
import { EventsPage } from './pages/EventsPage'
import { LecturesPage } from './pages/LecturesPage'
import { SourcedEventsPage } from './pages/SourcedEventsPage'

export function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<OverviewPage />} />
        <Route path="lecturers" element={<LecturersPage />} />
        <Route path="locations" element={<LocationsPage />} />
        <Route path="events" element={<EventsPage />} />
        <Route path="lectures" element={<LecturesPage />} />
        <Route path="sourced-events" element={<SourcedEventsPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  )
}

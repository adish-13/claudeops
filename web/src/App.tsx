import { Routes, Route, Navigate } from "react-router-dom";
import { Layout } from "./components/Layout";
import { HomePage } from "./pages/Home";
import { EpicPage } from "./pages/Epic";
import { NewEpicPage } from "./pages/NewEpic";
import { WorkspacePage } from "./pages/Workspace";
import { NewWorkspacePage } from "./pages/NewWorkspace";
import { SessionsPage } from "./pages/Sessions";
import { TranscriptPage } from "./pages/Transcript";
import { DebugPage } from "./pages/Debug";

export function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<HomePage />} />
        <Route path="/epics/new" element={<NewEpicPage />} />
        <Route path="/epics/:slug" element={<EpicPage />} />
        <Route path="/epics/:slug/workspaces/new" element={<NewWorkspacePage />} />
        <Route path="/epics/:slug/workspaces/:wsslug" element={<WorkspacePage />} />
        <Route path="/sessions" element={<SessionsPage />} />
        <Route path="/sessions/:id" element={<TranscriptPage />} />
        <Route path="/debug" element={<DebugPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}

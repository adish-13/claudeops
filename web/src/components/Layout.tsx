import { Outlet } from "react-router-dom";
import { Sidebar } from "./Sidebar";

export function Layout() {
  return (
    <div className="grid h-screen overflow-hidden" style={{ gridTemplateColumns: "280px 1fr" }}>
      <Sidebar />
      <Outlet />
    </div>
  );
}

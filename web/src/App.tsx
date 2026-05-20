import { Navigate, Route, Routes } from "react-router";

import { Toaster } from "@/components/ui/sonner";
import { Admin } from "@/pages/Admin";
import { Guest } from "@/pages/Guest";

// App splits the two surfaces of the plugin:
//   /admin*    operator-only admin console
//   /p/:token  recipient guest-pass landing page
// Everything else redirects to /admin so the deploy doesn't 404 on
// stray URLs.
export function App() {
  return (
    <>
      <Routes>
        <Route path="/admin/*" element={<Admin />} />
        <Route path="/admin" element={<Admin />} />
        <Route path="/p/:token" element={<Guest />} />
        <Route path="*" element={<Navigate to="/admin" replace />} />
      </Routes>
      <Toaster />
    </>
  );
}

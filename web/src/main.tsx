import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router";

import { App } from "@/App";
import { captureTokenFromURL } from "@/lib/authToken";
import "./index.css";

// Strip any ?token=... from the URL before React mounts so it doesn't
// leak via referrers or screen sharing. The captured token is held in
// JS memory only and attached to subsequent fetches by lib/api.
captureTokenFromURL();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </StrictMode>,
);

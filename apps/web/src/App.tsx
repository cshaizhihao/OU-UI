import { useEffect, useState } from "react";
import {
  clearStoredToken,
  getStoredToken,
  loadDashboard,
  login,
  type DashboardDTO,
  type SessionUser
} from "./api";
import { Shell } from "./components/Shell";
import { DashboardPage } from "./pages/DashboardPage";
import { LoginPage } from "./pages/LoginPage";

export function App() {
  const [dashboard, setDashboard] = useState<DashboardDTO | null>(null);
  const [user, setUser] = useState<SessionUser | null>(null);
  const [loading, setLoading] = useState(Boolean(getStoredToken()));
  const [loginLoading, setLoginLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!getStoredToken()) {
      return;
    }
    void refreshDashboard();
  }, []);

  async function refreshDashboard() {
    setLoading(true);
    setError("");
    try {
      setDashboard(await loadDashboard());
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to load dashboard");
      if (!getStoredToken()) {
        setUser(null);
      }
    } finally {
      setLoading(false);
    }
  }

  async function handleLogin(username: string, password: string) {
    setLoginLoading(true);
    setError("");
    try {
      const session = await login(username, password);
      setUser(session.user ?? { id: username, username, role: "operator" });
      setDashboard(await loadDashboard());
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
      clearStoredToken();
    } finally {
      setLoginLoading(false);
    }
  }

  function handleLogout() {
    clearStoredToken();
    setDashboard(null);
    setUser(null);
  }

  if (!getStoredToken()) {
    return <LoginPage error={error} loading={loginLoading} onLogin={handleLogin} />;
  }

  return (
    <Shell onLogout={handleLogout} user={user}>
      <DashboardPage
        data={dashboard}
        error={error}
        loading={loading}
        onRefresh={refreshDashboard}
      />
    </Shell>
  );
}

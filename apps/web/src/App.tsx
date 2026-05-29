import { Shell } from "./components/Shell";
import { DashboardPage } from "./pages/DashboardPage";
import { LoginPage } from "./pages/LoginPage";

export function App() {
  return (
    <>
      <LoginPage />
      <Shell>
        <DashboardPage />
      </Shell>
    </>
  );
}

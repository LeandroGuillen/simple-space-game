import { Navigate, Route, Router } from "@solidjs/router";

import { AppShell } from "./components/AppShell";
import { ColoniesPage } from "./routes/ColoniesPage";
import { FleetsPage } from "./routes/FleetsPage";
import { GalaxyPage } from "./routes/GalaxyPage";
import { PlanetPage } from "./routes/PlanetPage";
import { SetupPage } from "./routes/SetupPage";
import { TechnologyPage } from "./routes/TechnologyPage";
import { AppProvider, useAppContext } from "./state/app-context";

function HomeRoute() {
  const { state } = useAppContext();
  return (
    <Navigate
      href={
        state.session.serverUrl
          ? state.session.selectedGameId
            ? "/galaxy"
            : "/setup"
          : "/setup"
      }
    />
  );
}

export default function App() {
  return (
    <AppProvider>
      <Router root={AppShell}>
        <Route path="/" component={HomeRoute} />
        <Route path="/setup" component={SetupPage} />
        <Route path="/galaxy" component={GalaxyPage} />
        <Route path="/stars/:starId/planets/:planetId" component={PlanetPage} />
        <Route path="/colonies" component={ColoniesPage} />
        <Route path="/fleets" component={FleetsPage} />
        <Route path="/technology" component={TechnologyPage} />
        <Route path="*404" component={() => <Navigate href="/" />} />
      </Router>
    </AppProvider>
  );
}

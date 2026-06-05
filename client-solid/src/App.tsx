import { Navigate, Route, Router } from "@solidjs/router";

import { AppShell } from "./components/AppShell";
import { ColoniesPage } from "./routes/ColoniesPage";
import { FleetsPage } from "./routes/FleetsPage";
import { GalaxyPage } from "./routes/GalaxyPage";
import { MutationPage } from "./routes/MutationPage";
import { PlanetPage } from "./routes/PlanetPage";
import { RaceEditorPage } from "./routes/RaceEditorPage";
import { SetupPage } from "./routes/SetupPage";
import { ShipEditorPage } from "./routes/ShipEditorPage";
import { TechnologyPage } from "./routes/TechnologyPage";
import { AppProvider } from "./state/app-context";

export default function App() {
  return (
    <AppProvider>
      <Router root={AppShell}>
        <Route path="/" component={SetupPage} />
        <Route path="/race-editor" component={RaceEditorPage} />
        <Route path="/ship-editor" component={ShipEditorPage} />
        <Route path="/setup" component={() => <Navigate href="/" />} />
        <Route path="/setup/races" component={() => <Navigate href="/race-editor" />} />
        <Route path="/setup/custom-race" component={() => <Navigate href="/race-editor" />} />
        <Route path="/setup/ship-editor" component={() => <Navigate href="/ship-editor" />} />
        <Route path="/galaxy" component={GalaxyPage} />
        <Route path="/stars/:starId/planets/:planetId" component={PlanetPage} />
        <Route path="/colonies" component={ColoniesPage} />
        <Route path="/fleets" component={FleetsPage} />
        <Route path="/technology" component={TechnologyPage} />
        <Route path="/mutation" component={MutationPage} />
        <Route path="*404" component={() => <Navigate href="/" />} />
      </Router>
    </AppProvider>
  );
}

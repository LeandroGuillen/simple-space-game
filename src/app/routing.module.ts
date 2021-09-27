import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

import { ColoniesPageComponent } from './pages/colonies-page/colonies-page.component';
import { GamePageComponent } from './pages/game-page/game-page.component';
import { PlanetViewComponent } from './pages/planet-view/planet-view.component';
import { ServerPageComponent } from './pages/server-page/server-page.component';
import { StarListPageComponent } from './pages/star-list-page/star-list-page.component';
import { StarViewComponent } from './pages/star-view/star-view.component';

const routes: Routes = [
  { path: 'server', component: ServerPageComponent },
  { path: 'game', component: GamePageComponent },
  { path: 'stars/:star_id', component: StarViewComponent },
  { path: 'stars/:star_id/planets/:planet_id', component: PlanetViewComponent },
  { path: 'stars', component: StarListPageComponent },
  { path: 'colonies', component: ColoniesPageComponent },
  { path: '**', redirectTo: '/home', },
];

@NgModule({
  imports: [ RouterModule.forRoot(routes) ],
  exports: [ RouterModule ]
})
export class RoutingModule {
}

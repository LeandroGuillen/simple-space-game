import {NgModule} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {StarListPageComponent} from "./star-list-page/star-list-page.component";
import {StarViewComponent} from "./star-view/star-view.component";
import {PlanetViewComponent} from "./planet-view/planet-view.component";
import {ServerPageComponent} from "./server-page/server-page.component";
import {GamePageComponent} from "./game-page/game-page.component";
import {ColoniesPageComponent} from "./colonies-page/colonies-page.component";

const routes: Routes = [
  {path: 'server', component: ServerPageComponent},
  {path: 'game', component: GamePageComponent},
  {path: 'stars/:star_id', component: StarViewComponent},
  {path: 'stars/:star_id/planets/:planet_id', component: PlanetViewComponent},
  {path: 'stars', component: StarListPageComponent},
  {path: 'colonies', component: ColoniesPageComponent},
  {path: '**', redirectTo: '/home',},
];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule]
})
export class RoutingModule {
}

import {NgModule} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {GalaxyListComponent} from "./galaxy-list/galaxy-list.component";
import {StarViewComponent} from "./star-view/star-view.component";
import {PlanetViewComponent} from "./planet-view/planet-view.component";
import {HomePageComponent} from "./home-page/home-page.component";
import {GamePageComponent} from "./game-page/game-page.component";

const routes: Routes = [
  {path: 'home', component: HomePageComponent},
  {path: 'game', component: GamePageComponent},
  {path: 'stars/:star_id', component: StarViewComponent},
  {path: 'stars/:star_id/planets/:planet_id', component: PlanetViewComponent},
  {path: 'galaxy', component: GalaxyListComponent},
];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule]
})
export class AppRoutingModule {
}

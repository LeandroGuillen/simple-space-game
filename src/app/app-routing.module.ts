import {NgModule} from '@angular/core';
import {RouterModule, Routes} from '@angular/router';
import {GalaxyListComponent} from "./galaxy-list/galaxy-list.component";
import {StarViewComponent} from "./star-view/star-view.component";
import {PlanetViewComponent} from "./planet-view/planet-view.component";

const routes: Routes = [
  {path: 'stars/:name', component: StarViewComponent},
  {path: 'planets/:name', component: PlanetViewComponent},
  {path: '**', component: GalaxyListComponent},
];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule]
})
export class AppRoutingModule {
}

import {HttpClientModule} from '@angular/common/http';
import {NgModule} from '@angular/core';
import {BrowserModule} from '@angular/platform-browser';
import {BrowserAnimationsModule} from '@angular/platform-browser/animations';

import {AppRoutingModule} from './app-routing.module';
import {AppComponent} from './app.component';
import {MaterialModule} from './material.module';
import {GalaxyListComponent} from "./galaxy-list/galaxy-list.component";
import {StarViewComponent} from "./star-view/star-view.component";
import {PlanetViewComponent} from "./planet-view/planet-view.component";
import {ReactiveFormsModule} from "@angular/forms";
import {HomePageComponent} from "./home-page/home-page.component";
import {PlanetIconComponent} from "./planet-view/planet-icon/planet-icon.component";
import {NewGameDialogComponent} from "./galaxy-list/new-game-dialog/new-game-dialog.component";
import { GamePageComponent } from './game-page/game-page.component';

@NgModule({
  declarations: [
    AppComponent,
    GalaxyListComponent,
    StarViewComponent,
    PlanetViewComponent,
    HomePageComponent,
    PlanetIconComponent,
    NewGameDialogComponent,
    GamePageComponent
  ],
  imports: [
    BrowserModule,
    BrowserAnimationsModule,
    AppRoutingModule,
    HttpClientModule,
    BrowserAnimationsModule,
    MaterialModule,
    ReactiveFormsModule,
  ],
  providers: [],
  bootstrap: [AppComponent]
})
export class AppModule {
}

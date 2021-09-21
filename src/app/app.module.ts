import {HttpClientModule} from '@angular/common/http';
import {NgModule} from '@angular/core';
import {BrowserModule} from '@angular/platform-browser';
import {BrowserAnimationsModule} from '@angular/platform-browser/animations';

import {AppRoutingModule} from './app-routing.module';
import {AppComponent} from './app.component';
import {MaterialModule} from './material.module';
import {StarListPageComponent} from "./star-list-page/star-list-page.component";
import {StarViewComponent} from "./star-view/star-view.component";
import {PlanetViewComponent} from "./planet-view/planet-view.component";
import {ReactiveFormsModule} from "@angular/forms";
import {HomePageComponent} from "./home-page/home-page.component";
import {PlanetIconComponent} from "./planet-view/planet-icon/planet-icon.component";
import {NewGameDialogComponent} from "./game-page/new-game-dialog/new-game-dialog.component";
import {GamePageComponent} from './game-page/game-page.component';
import {ColoniesPageComponent} from './colonies-page/colonies-page.component';
import {GalaxyViewComponent} from './galaxy-view/galaxy-view.component';

@NgModule({
  declarations: [
    AppComponent,
    StarListPageComponent,
    StarViewComponent,
    PlanetViewComponent,
    HomePageComponent,
    PlanetIconComponent,
    NewGameDialogComponent,
    GamePageComponent,
    ColoniesPageComponent,
    GalaxyViewComponent,
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

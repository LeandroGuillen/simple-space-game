import {HttpClientModule} from '@angular/common/http';
import {NgModule} from '@angular/core';
import {BrowserModule} from '@angular/platform-browser';
import {BrowserAnimationsModule} from '@angular/platform-browser/animations';

import {RoutingModule} from './routing.module';
import {AppComponent} from './app.component';
import {MaterialModule} from './material.module';
import {StarListPageComponent} from "./star-list-page/star-list-page.component";
import {StarViewComponent} from "./star-view/star-view.component";
import {PlanetViewComponent} from "./planet-view/planet-view.component";
import {ReactiveFormsModule} from "@angular/forms";
import {ServerPageComponent} from "./server-page/server-page.component";
import {PlanetIconComponent} from "./planet-view/planet-icon/planet-icon.component";
import {NewGameDialogComponent} from "./game-page/new-game-dialog/new-game-dialog.component";
import {GamePageComponent} from './game-page/game-page.component';
import {ColoniesPageComponent} from './colonies-page/colonies-page.component';
import {GalaxyViewComponent} from './galaxy-view/galaxy-view.component';
import {TopBarComponent} from './top-bar/top-bar.component';
import {ChangeBuildingDialogComponent} from "./colonies-page/change-building-dialog/change-building-dialog.component";
import {StoreModule} from '@ngrx/store';
import {counterReducer} from "./counter.reducer";
import {MyCounterComponent} from "./my-counter.component";

@NgModule({
  declarations: [
    AppComponent,
    StarListPageComponent,
    StarViewComponent,
    PlanetViewComponent,
    ServerPageComponent,
    PlanetIconComponent,
    NewGameDialogComponent,
    GamePageComponent,
    ColoniesPageComponent,
    GalaxyViewComponent,
    TopBarComponent,
    ChangeBuildingDialogComponent,
    MyCounterComponent
  ],
  imports: [
    BrowserModule,
    BrowserAnimationsModule,
    RoutingModule,
    HttpClientModule,
    BrowserAnimationsModule,
    MaterialModule,
    ReactiveFormsModule,
    StoreModule.forRoot({count: counterReducer}),
  ],
  providers: [],
  bootstrap: [AppComponent]
})
export class AppModule {
}

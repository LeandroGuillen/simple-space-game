import { HttpClientModule } from '@angular/common/http';
import { NgModule } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { EffectsModule } from '@ngrx/effects';
import { StoreModule } from '@ngrx/store';

import { AppComponent } from './app.component';
import {
  ChangeBuildingDialogComponent,
} from './colonies-page/change-building-dialog/change-building-dialog.component';
import { ColoniesPageComponent } from './colonies-page/colonies-page.component';
import { GalaxyViewComponent } from './galaxy-view/galaxy-view.component';
import { GamePageComponent } from './game-page/game-page.component';
import { NewGameDialogComponent } from './game-page/new-game-dialog/new-game-dialog.component';
import { MaterialModule } from './material.module';
import { PlanetIconComponent } from './planet-view/planet-icon/planet-icon.component';
import { PlanetViewComponent } from './planet-view/planet-view.component';
import { RoutingModule } from './routing.module';
import { ServerPageComponent } from './server-page/server-page.component';
import { StarListPageComponent } from './star-list-page/star-list-page.component';
import { StarViewComponent } from './star-view/star-view.component';
import { GameEffects } from './store/effects';
import { gameReducer } from './store/reducers';
import { TopBarComponent } from './top-bar/top-bar.component';

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
  ],
  imports: [
    BrowserModule,
    BrowserAnimationsModule,
    RoutingModule,
    HttpClientModule,
    BrowserAnimationsModule,
    MaterialModule,
    ReactiveFormsModule,
    StoreModule.forRoot({ game: gameReducer }),
    EffectsModule.forRoot([ GameEffects ])
  ],
  providers: [],
  bootstrap: [ AppComponent ]
})
export class AppModule {
}

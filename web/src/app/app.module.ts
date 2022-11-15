import { HttpClientModule } from '@angular/common/http';
import { NgModule } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { EffectsModule } from '@ngrx/effects';
import { routerReducer, StoreRouterConnectingModule } from '@ngrx/router-store';
import { StoreModule } from '@ngrx/store';
import { StoreDevtoolsModule } from '@ngrx/store-devtools';

import { environment } from '../environments/environment';
import { AppComponent } from './app.component';
import { ComponentsModule } from './components/components.module';
import { MaterialModule } from './material.module';
import { ChangeBuildingDialogComponent } from './pages/colonies-page/change-building-dialog/change-building-dialog.component';
import { ColoniesPageComponent } from './pages/colonies-page/colonies-page.component';
import { EditShipDesignDialogComponent } from './pages/fleets-page/edit-ship-design-dialog/edit-ship-design-dialog.component';
import { FleetTableComponent } from './pages/fleets-page/fleet-table/fleet-table.component';
import { FleetsPageComponent } from './pages/fleets-page/fleets-page.component';
import { ShipDesignTableComponent } from './pages/fleets-page/ship-design-table/ship-design-table.component';
import { ShipDesignComponent } from './pages/fleets-page/ship-design/ship-design.component';
import { GamePageComponent } from './pages/game-page/game-page.component';
import { NewGameDialogComponent } from './pages/game-page/new-game-dialog/new-game-dialog.component';
import { PlanetViewComponent } from './pages/planet-view/planet-view.component';
import { NewServerDialogComponent } from './pages/server-page/new-server-dialog/new-server-dialog.component';
import { ServerListItemComponent } from './pages/server-page/server-list-item/server-list-item.component';
import { ServerPageComponent } from './pages/server-page/server-page.component';
import { ServerStatusIconComponent } from './pages/server-page/server-status-icon/server-status-icon.component';
import { StarListPageComponent } from './pages/star-list-page/star-list-page.component';
import { StarViewComponent } from './pages/star-view/star-view.component';
import { RoutesModule } from './routes.module';
import { GameEffects } from './store/effects';
import { CustomSerializer } from './store/reducer-router';
import { gameReducer } from './store/reducers';

@NgModule({
  declarations: [
    AppComponent,
    StarListPageComponent,
    StarViewComponent,
    PlanetViewComponent,
    ServerPageComponent,
    NewGameDialogComponent,
    NewServerDialogComponent,
    GamePageComponent,
    ColoniesPageComponent,
    ChangeBuildingDialogComponent,
    FleetsPageComponent,
    FleetTableComponent,
    ShipDesignTableComponent,
    ShipDesignComponent,
    EditShipDesignDialogComponent,
    ServerStatusIconComponent,
    ServerListItemComponent,
  ],
  imports: [
    // angular
    BrowserModule,
    BrowserAnimationsModule,
    RoutesModule,
    HttpClientModule,
    BrowserAnimationsModule,
    MaterialModule,
    ReactiveFormsModule,
    // ngrx
    StoreModule.forRoot({ game: gameReducer, router: routerReducer }),
    EffectsModule.forRoot([GameEffects]),
    StoreDevtoolsModule.instrument({
      maxAge: 25,
      logOnly: environment.production,
    }),
    StoreRouterConnectingModule.forRoot({ serializer: CustomSerializer }),
    // other
    ComponentsModule,
  ],
  providers: [],
  bootstrap: [AppComponent],
})
export class AppModule {}

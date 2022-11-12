import { NgModule } from '@angular/core';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { BrowserModule } from '@angular/platform-browser';

import { MaterialModule } from '../material.module';
import { RoutesModule } from '../routes.module';
import { SetTaxDialogComponent } from './dialogs/set-tax-dialog/set-tax-dialog.component';
import { GalaxyViewComponent } from './galaxy-view/galaxy-view.component';
import { KpiBarComponent } from './kpi-bar/kpi-bar.component';
import { KpiComponent } from './kpi-bar/kpi/kpi.component';
import { PlanetIconComponent } from './planet-icon/planet-icon.component';
import { PlanetMiniViewComponent } from './planet-mini-view/planet-mini-view.component';
import { RichSelectComponent } from './rich-select/rich-select.component';
import { StarIconComponent } from './star-icon/star-icon.component';
import { TopBarComponent } from './top-bar/top-bar.component';

const components = [
    GalaxyViewComponent,
    TopBarComponent,
    StarIconComponent,
    KpiComponent,
    KpiBarComponent,
    RichSelectComponent,
    PlanetMiniViewComponent,
    PlanetIconComponent,
    SetTaxDialogComponent,
];

@NgModule({
    declarations: components,
    exports: components,
    imports: [
        BrowserModule,
        MaterialModule,
        RoutesModule,
        ReactiveFormsModule,
        FormsModule,
    ],
})
export class ComponentsModule {}

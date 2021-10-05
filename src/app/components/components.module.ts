import { NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';

import { MaterialModule } from '../material.module';
import { RoutingModule } from '../routing.module';
import { GalaxyViewComponent } from './galaxy-view/galaxy-view.component';
import { PlanetIconComponent } from './planet-icon/planet-icon.component';
import { PlanetMiniViewComponent } from './planet-mini-view/planet-mini-view.component';
import { RichSelectComponent } from './rich-select/rich-select.component';
import { StarIconComponent } from './star-icon/star-icon.component';
import { KpiComponent } from './top-bar/kpi/kpi.component';
import { TopBarComponent } from './top-bar/top-bar.component';

const components = [
	GalaxyViewComponent,
	TopBarComponent,
	StarIconComponent,
	KpiComponent,
	RichSelectComponent,
	PlanetMiniViewComponent,
	PlanetIconComponent,
]

@NgModule({
	declarations: components,
	exports: components,
	imports: [ BrowserModule, MaterialModule, RoutingModule ],
})
export class ComponentsModule {
}

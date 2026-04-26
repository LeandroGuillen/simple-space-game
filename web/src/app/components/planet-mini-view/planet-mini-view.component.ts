import { CdkDragDrop, moveItemInArray } from '@angular/cdk/drag-drop';
import { Component, Input } from '@angular/core';
import { MatDialog, MatDialogRef } from '@angular/material/dialog';
import { Store } from '@ngrx/store';

import { Planet } from '../../model/model';
import {
  ChangeBuildingDialogComponent,
} from '../../pages/colonies-page/change-building-dialog/change-building-dialog.component';
import { StringService } from '../../services/string.service';
import { changeBuildingRequest } from '../../store/actions';
import { AppState } from '../../store/reducers';

@Component({
  selector: 'ssg-planet-mini-view',
  standalone: false,
  templateUrl: './planet-mini-view.component.html',
  styleUrls: [ './planet-mini-view.component.scss' ]
})
export class PlanetMiniViewComponent {
  shortDescription = '';
  private _planet!: Planet;

  constructor(
    private readonly store: Store<AppState>,
    private readonly strings: StringService,
    private readonly dialog: MatDialog) { }

  @Input() set planet(p: Planet) {
    this._planet = p;
    this.shortDescription = this.strings.getPlanetShortDescription(p);
  }

  get planet(): Planet {
    return this._planet;
  }

  drop(event: Event) {
    const ddEvent = event as unknown as CdkDragDrop<string[]>;
    moveItemInArray(this.planet.buildings, ddEvent.previousIndex, ddEvent.currentIndex);
  }

  changeBuilding(planet: Planet) {
    const dialogRef: MatDialogRef<ChangeBuildingDialogComponent, number> =
      this.dialog.open(ChangeBuildingDialogComponent);

    dialogRef.afterClosed().subscribe(buildingID => {
      if (!buildingID) {
        return;
      }
      this.store.dispatch(changeBuildingRequest({
        planetID: planet.id,
        buildingID
      }))
    });
  }
}

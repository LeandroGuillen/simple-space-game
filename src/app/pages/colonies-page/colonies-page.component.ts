import { AfterViewInit, Component, OnInit, ViewChild } from '@angular/core';
import { MatDialog, MatDialogRef } from '@angular/material/dialog';
import { MatSort } from '@angular/material/sort';
import { MatTableDataSource } from '@angular/material/table';
import { Store } from '@ngrx/store';

import { Planet } from '../../model/model';
import { changeBuildingRequest } from '../../store/actions';
import { AppState } from '../../store/reducers';
import { selectColonies } from '../../store/selectors';
import {
  ChangeBuildingDialogComponent,
} from './change-building-dialog/change-building-dialog.component';

const planetData: Planet[] = []

@Component({
  selector: 'crn-colonies-page',
  templateUrl: './colonies-page.component.html',
  styleUrls: [ './colonies-page.component.scss' ]
})
export class ColoniesPageComponent implements AfterViewInit, OnInit {
  displayedColumns: (keyof Planet | 'actions')[] = [ 'id', 'name', 'farmers', 'workers', 'scientists', 'current_project', 'actions' ];
  dataSource = new MatTableDataSource(planetData);

  colonyList!: Planet[];

  @ViewChild(MatSort) sort!: MatSort;

  constructor(
    private readonly store: Store<AppState>,
    private readonly dialog: MatDialog,) {
  }

  ngOnInit() {
    this.store.select(selectColonies).subscribe(colonies => {
      this.colonyList = colonies;
      this.dataSource = new MatTableDataSource(colonies);
      this.dataSource.sort = this.sort;
    })
  }

  ngAfterViewInit() {
    this.dataSource.sort = this.sort;
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

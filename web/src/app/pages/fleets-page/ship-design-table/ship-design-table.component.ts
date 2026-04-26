import { Component, OnInit, ViewChild } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { MatSort } from '@angular/material/sort';
import { MatTableDataSource } from '@angular/material/table';
import { Store } from '@ngrx/store';
import { take } from 'rxjs/operators';

import { ShipDesign } from '../../../model/combat';
import { AppState } from '../../../store/reducers';
import {
  selectAllShipDesigns,
  selectShipDesignByName,
} from '../../../store/selectors/player.selectors';
import {
  EditShipDesignDialogComponent,
} from '../edit-ship-design-dialog/edit-ship-design-dialog.component';

type ShipDesignRow = {
  hullsize: string;
  name: string;
  isCommon: 'Yes' | 'No';
};

@Component({
  selector: 'ssg-ship-design-table',
  standalone: false,
  templateUrl: './ship-design-table.component.html',
  styleUrls: ['./ship-design-table.component.scss'],
})
export class ShipDesignTableComponent implements OnInit {
  dataSource!: MatTableDataSource<ShipDesignRow>;

  columns = [
    { key: 'name', name: 'Name' },
    { key: 'hullsize', name: 'Hull size' },
    { key: 'isCommon', name: 'Is Common' },
  ];
  displayedColumns = this.columns.map(c => c.key);

  @ViewChild(MatSort) sort!: MatSort;

  constructor(
    private readonly store: Store<AppState>,
    private readonly dialog: MatDialog
  ) {}

  ngOnInit(): void {
    this.store.select(selectAllShipDesigns).subscribe(shipDesigns => {
      if (!shipDesigns) {
        return;
      }
      this.dataSource = new MatTableDataSource(
        shipDesigns.map(ShipDesignTableComponent.mapShipDesign2Row)
      );
      this.dataSource.sort = this.sort;
    });
  }

  selectRow(row: ShipDesignRow) {
    this.store
      .select(selectShipDesignByName(row.name))
      .pipe(take(1))
      .subscribe(design => {
        this.dialog.open(EditShipDesignDialogComponent, {
          data: {
            shipDesignName: row.name,
            design,
          },
        });
      });
  }

  newDesign() {
    this.dialog.open(EditShipDesignDialogComponent);
  }

  private static mapShipDesign2Row(s: ShipDesign): ShipDesignRow {
    return {
      hullsize: s.hullSize.name,
      name: s.name,
      isCommon: s.isCommon ? 'Yes' : 'No',
    };
  }
}

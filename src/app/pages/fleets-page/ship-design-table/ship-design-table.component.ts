import { Component, OnInit, ViewChild } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { MatSort } from '@angular/material/sort';
import { MatTableDataSource } from '@angular/material/table';
import { Store } from '@ngrx/store';
import { tap } from 'rxjs/operators';

import { AppState } from '../../../store/reducers';
import { selectShipDesigns } from '../../../store/selectors/player.selectors';
import { EditShipDesignDialogComponent } from '../edit-ship-design-dialog/edit-ship-design-dialog.component';

type ShipDesignRow = {
  hullsize: string;
  name: string;
};

@Component({
  selector: 'ssg-ship-design-table',
  templateUrl: './ship-design-table.component.html',
  styleUrls: ['./ship-design-table.component.scss'],
})
export class ShipDesignTableComponent implements OnInit {
  dataSource!: MatTableDataSource<ShipDesignRow>;

  columns = [
    { key: 'name', name: 'Name' },
    { key: 'hullsize', name: 'Hull size' },
  ];
  displayedColumns = this.columns.map(c => c.key);

  @ViewChild(MatSort) sort!: MatSort;

  constructor(
    private readonly store: Store<AppState>,
    private readonly dialog: MatDialog
  ) {}

  ngOnInit(): void {
    this.store
      .select(selectShipDesigns)
      .pipe(
        tap(shipDesigns => {
          if (!shipDesigns) {
            return;
          }
          this.dataSource = new MatTableDataSource(
            shipDesigns.map(s => ({ hullsize: s.hullSize.name, name: s.name }))
          );
        })
      )
      .subscribe();
  }

  ngAfterViewInit() {
    this.dataSource.sort = this.sort;
  }

  selectRow(row: ShipDesignRow) {
    this.dialog.open(EditShipDesignDialogComponent, {
      data: {
        shipDesignName: row.name,
      },
    });
  }
}

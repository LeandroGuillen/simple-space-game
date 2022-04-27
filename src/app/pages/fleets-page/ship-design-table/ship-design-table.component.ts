import { Component, OnInit, ViewChild } from '@angular/core';
import { MatSort } from '@angular/material/sort';
import { MatTableDataSource } from '@angular/material/table';
import { Store } from '@ngrx/store';
import { tap } from 'rxjs/operators';

import { AppState } from '../../../store/reducers';
import { selectShipDesigns } from '../../../store/selectors/player.selectors';

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
    { key: 'hullsize', name: 'Hull size' },
    { key: 'name', name: 'Name' },
  ];
  displayedColumns = this.columns.map(c => c.key);

  @ViewChild(MatSort) sort!: MatSort;

  constructor(private readonly store: Store<AppState>) {}

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
}

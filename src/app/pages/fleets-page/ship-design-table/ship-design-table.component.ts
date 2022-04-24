import { Component, OnInit, ViewChild } from '@angular/core';
import { MatSort } from '@angular/material/sort';
import { MatTableDataSource } from '@angular/material/table';
import { Store } from '@ngrx/store';

import { AppState } from '../../../store/reducers';
import { selectShipDesigns } from '../../../store/selectors/player.selectors';

@Component({
  selector: 'ssg-ship-design-table',
  templateUrl: './ship-design-table.component.html',
  styleUrls: ['./ship-design-table.component.scss'],
})
export class ShipDesignTableComponent implements OnInit {
  dataSource: any; //MatTableDataSource;

  columns = [
    { key: 'hullsize', name: 'Hull size' },
    { key: 'name', name: 'Name' },
  ];
  displayedColumns = this.columns.map(c => c.key);

  @ViewChild(MatSort) sort!: MatSort;

  constructor(private readonly store: Store<AppState>) {}

  ngOnInit(): void {
    this.dataSource = new MatTableDataSource([
      { hullsize: 'Frigate', name: 'Scout' },
      { hullsize: 'Frigate', name: 'Bomber' },
      { hullsize: 'Destroyer', name: 'Missile Boat' },
      { hullsize: 'Frigate', name: 'Phaser' },
    ]);

    this.store.select(selectShipDesigns).subscribe(sdList => {
      console.log(1, sdList);
    });
  }

  ngAfterViewInit() {
    this.dataSource.sort = this.sort;
  }
}

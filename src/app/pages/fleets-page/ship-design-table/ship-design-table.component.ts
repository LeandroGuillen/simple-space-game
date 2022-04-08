import { Component, OnInit, ViewChild } from '@angular/core';
import { MatSort } from '@angular/material/sort';
import { MatTableDataSource } from '@angular/material/table';

@Component({
  selector: 'ssg-ship-design-table',
  templateUrl: './ship-design-table.component.html',
  styleUrls: ['./ship-design-table.component.scss'],
})
export class ShipDesignTableComponent implements OnInit {
  dataSource: any; //MatTableDataSource;

  columns = [
    { key: 'id', name: 'ID' },
    { key: 'hullsize', name: 'Hull size' },
    { key: 'name', name: 'Name' },
  ];
  displayedColumns = this.columns.map((c) => c.key);

  @ViewChild(MatSort) sort!: MatSort;

  ngOnInit(): void {
    this.dataSource = new MatTableDataSource([
      { id: 1, hullsize: 'Frigate', name: 'Scout' },
      { id: 2, hullsize: 'Frigate', name: 'Bomber' },
      { id: 3, hullsize: 'Destroyer', name: 'Missile Boat' },
      { id: 4, hullsize: 'Frigate', name: 'Phaser' },
    ]);
  }
  ngAfterViewInit() {
    this.dataSource.sort = this.sort;
  }
}

import { Component, OnInit, ViewChild } from '@angular/core';
import { MatSort } from '@angular/material/sort';
import { MatTableDataSource } from '@angular/material/table';
import { Store } from '@ngrx/store';

import { Galaxy, Star } from '../../model/model';
import { AppState } from '../../store/reducers';
import { selectGalaxy } from '../../store/selectors';

const starData: Star[] = [];

@Component({
  selector: 'crn-galaxy-list',
  templateUrl: './star-list-page.component.html',
  styleUrls: [ './star-list-page.component.scss' ]
})
export class StarListPageComponent implements OnInit {
  displayedColumns: (keyof Star)[] = [ 'id', 'name', 'position', 'color', 'numPlanets' ];
  dataSource = new MatTableDataSource(starData);

  galaxy!: Galaxy;

  @ViewChild(MatSort) sort!: MatSort;

  constructor(private readonly store: Store<AppState>) {
  }

  ngOnInit() {
    this.store.select(selectGalaxy).subscribe(galaxy => {
      if (!galaxy) {
        return;
      }
      this.galaxy = galaxy;
      this.dataSource = new MatTableDataSource(galaxy.stars);
      this.dataSource.sort = this.sort;
    });
  }

  ngAfterViewInit() {
    this.dataSource.sort = this.sort;
  }
}

import { Component, OnInit, ViewChild } from '@angular/core';
import { FormBuilder, FormGroup } from '@angular/forms';
import { MatSort } from '@angular/material/sort';
import { MatTableDataSource } from '@angular/material/table';
import { Store } from '@ngrx/store';

import { Galaxy, Star } from '../../model/model';
import { AppState } from '../../store/reducers';
import { selectGalaxy } from '../../store/selectors';

const starData: Star[] = [];

@Component({
  selector: 'ssg-galaxy-list',
  templateUrl: './star-list-page.component.html',
  styleUrls: [ './star-list-page.component.scss' ]
})
export class StarListPageComponent implements OnInit {
  displayedColumns: (keyof Star)[] = [ 'id', 'name', 'position', 'color', 'numPlanets' ];
  dataSource = new MatTableDataSource(starData);

  galaxy!: Galaxy;
  form!: FormGroup;
  showTable!: boolean;

  @ViewChild(MatSort) sort!: MatSort;

  constructor(private readonly store: Store<AppState>,
    private readonly fb: FormBuilder) {
  }

  ngOnInit() {
    const showAsTable = this.getShowAsTable();
    this.showTable = showAsTable;
    this.form = this.fb.group({
      asTable: showAsTable
    });

    this.form.controls.asTable.valueChanges.subscribe(show => {
      this.setShowAsTable(show);
      this.showTable = show;
    });

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

  private getShowAsTable(): boolean {
    const show = window.localStorage.getItem('STAR-LIST-showAsTable') === 'true' || false;
    return show;
  }

  private setShowAsTable(show: boolean) {
    window.localStorage.setItem('STAR-LIST-showAsTable', show + '');
  }
}

import { Component, OnInit } from '@angular/core';
import { Store } from '@ngrx/store';

import { Galaxy } from '../model/model';
import { AppState } from '../store/reducers';
import { selectGalaxy } from '../store/selectors';

@Component({
  selector: 'crn-galaxy-list',
  templateUrl: './star-list-page.component.html',
  styleUrls: [ './star-list-page.component.scss' ]
})
export class StarListPageComponent implements OnInit {
  galaxy!: Galaxy;

  constructor(private readonly store: Store<AppState>) {
  }

  ngOnInit() {
    this.store.select(selectGalaxy).subscribe(galaxy => {
      if (!galaxy) {
        return;
      }
      this.galaxy = galaxy;
    });
  }
}

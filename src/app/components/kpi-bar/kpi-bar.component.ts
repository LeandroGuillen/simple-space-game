import { Component, OnInit } from '@angular/core';
import { Store } from '@ngrx/store';
import { Observable } from 'rxjs';
import { turnRequest } from 'src/app/store/actions';
import { AppState } from 'src/app/store/reducers';
import { selectPlayerTreasury } from 'src/app/store/selectors';

@Component({
  selector: 'ssg-kpi-bar',
  templateUrl: './kpi-bar.component.html',
  styleUrls: ['./kpi-bar.component.scss']
})
export class KpiBarComponent implements OnInit {
  treasury!: Observable<number | undefined>;

  constructor(
    private readonly store: Store<AppState>
  ) { }

  ngOnInit(): void {
    this.treasury = this.store.select(selectPlayerTreasury(1));
  }

  turn() {
    this.store.dispatch(turnRequest());
  }
}

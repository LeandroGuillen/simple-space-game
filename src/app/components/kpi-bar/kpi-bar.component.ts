import { Component, OnInit } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { Store } from '@ngrx/store';
import { Observable } from 'rxjs';
import { turnRequest } from 'src/app/store/actions';
import { AppState } from 'src/app/store/reducers';
import { selectTreasury } from 'src/app/store/selectors/player.selectors';

import { SetTaxDialogComponent } from '../dialogs/set-tax-dialog/set-tax-dialog.component';

@Component({
  selector: 'ssg-kpi-bar',
  templateUrl: './kpi-bar.component.html',
  styleUrls: ['./kpi-bar.component.scss']
})
export class KpiBarComponent implements OnInit {
  treasury!: Observable<number>;

  constructor(
    private readonly store: Store<AppState>,
    public dialog: MatDialog
  ) { }

  ngOnInit(): void {
    this.treasury = this.store.select(selectTreasury);
  }

  turn() {
    this.store.dispatch(turnRequest());
  }

  taxDialog(): void {
    this.dialog.open(SetTaxDialogComponent);
  }
}

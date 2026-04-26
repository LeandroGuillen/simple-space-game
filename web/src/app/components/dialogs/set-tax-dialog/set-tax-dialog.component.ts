import { Component, Inject, OnInit } from '@angular/core';
import { UntypedFormControl } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { Store } from '@ngrx/store';
import { tap } from 'rxjs/operators';
import { updateTaxRate } from 'src/app/store/actions';
import { AppState } from 'src/app/store/reducers';
import { selectTaxRate } from 'src/app/store/selectors/player.selectors';

@Component({
  selector: 'ssg-set-tax-dialog',
  standalone: false,
  templateUrl: './set-tax-dialog.component.html',
  styleUrls: ['./set-tax-dialog.component.scss'],
})
export class SetTaxDialogComponent implements OnInit {
  taxRateControl = new UntypedFormControl();

  constructor(
    public dialogRef: MatDialogRef<SetTaxDialogComponent>,
    private readonly store: Store<AppState>,
    @Inject(MAT_DIALOG_DATA) public data: any
  ) {}

  ngOnInit() {
    this.store
      .select(selectTaxRate)
      .pipe(
        tap((taxRate) => {
          this.taxRateControl.setValue(taxRate + '');
        })
      )
      .subscribe();

    this.taxRateControl.valueChanges.subscribe((newVal) => {
      const newTaxRate = parseInt(newVal);
      this.store.dispatch(updateTaxRate({ newTaxRate }));
    });
  }
}

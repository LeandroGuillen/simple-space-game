import { Component } from '@angular/core';
import { FormBuilder, FormGroup } from '@angular/forms';
import { MatDialogRef } from '@angular/material/dialog';

@Component({
  selector: 'ssg-change-building-dialog',
  templateUrl: './change-building-dialog.component.html',
  styleUrls: [ './change-building-dialog.component.scss' ]
})
export class ChangeBuildingDialogComponent {
  form: FormGroup;

  availableBuildings = [
    'Automated Factories',
    'Hydroponic Farms'
  ];

  constructor(fb: FormBuilder,
    readonly dialogRef: MatDialogRef<ChangeBuildingDialogComponent>
  ) {
    this.form = fb.group({
      selectedBuilding: '',
    });
  }

  ok(): void {
    if (this.form.invalid) {
      return;
    }
    this.dialogRef.close(this.form.value.selectedBuilding);
  }

  cancel(): void {
    this.dialogRef.close();
  }
}

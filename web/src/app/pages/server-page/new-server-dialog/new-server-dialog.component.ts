import { Component } from '@angular/core';
import { UntypedFormBuilder, UntypedFormGroup } from '@angular/forms';
import { MatDialogRef } from '@angular/material/dialog';

@Component({
  selector: 'ssg-new-server-dialog',
  standalone: false,
  templateUrl: './new-server-dialog.component.html',
  styleUrls: [ './new-server-dialog.component.scss' ]
})
export class NewServerDialogComponent {
  form: UntypedFormGroup;

  constructor(
    readonly fb: UntypedFormBuilder,
    public readonly dialogRef: MatDialogRef<NewServerDialogComponent>
  ) {
    this.form = fb.group({
      name: 'A Simple Space Game Server',
      url: 'http://localhost:8080'
    });
  }

  cancel(): void {
    this.dialogRef.close();
  }

  newServer(): void {
    if (this.form.invalid) {
      return;
    }
    this.dialogRef.close(this.form.value);
  }
}

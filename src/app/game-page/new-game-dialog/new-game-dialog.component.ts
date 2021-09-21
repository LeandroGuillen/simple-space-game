import {Component} from '@angular/core';
import {MatDialogRef} from "@angular/material/dialog";
import {FormBuilder, FormGroup} from "@angular/forms";

export interface DialogData {
  animal: string;
  name: string;
}

@Component({
  selector: 'crn-new-game-dialog',
  templateUrl: './new-game-dialog.component.html',
  styleUrls: ['./new-game-dialog.component.scss']
})
export class NewGameDialogComponent {
  form: FormGroup;

  galaxySizes = [
    {
      value: 3,
      label: 'Small',
    },
    {
      value: 10,
      label: 'Medium',
      disabled: true,
    },
    {
      value: 50,
      label: 'Large',
      disabled: true,
    }
  ];

  constructor(fb: FormBuilder,
              public readonly dialogRef: MatDialogRef<NewGameDialogComponent>
  ) {
    this.form = fb.group({
      numStars: this.galaxySizes[0].value,
      playerName: 'Augustus'
    });
  }

  cancel(): void {
    this.dialogRef.close();
  }

  newGame(): void {
    if (this.form.invalid) {
      return;
    }
    this.dialogRef.close(this.form.value.numStars);
  }

}

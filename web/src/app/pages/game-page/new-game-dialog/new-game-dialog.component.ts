import { Component } from '@angular/core';
import { FormBuilder, FormGroup } from '@angular/forms';
import { MatDialogRef } from '@angular/material/dialog';

import { NewGameRequest } from '../../../store/actions';

@Component({
  selector: 'ssg-new-game-dialog',
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

  constructor(
    readonly fb: FormBuilder,
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
    const form = this.form.value;
    const newGameRequest: NewGameRequest = {
      num_stars: form.numStars,
      player_name: form.playerName,
    }
    this.dialogRef.close(newGameRequest);
  }

}

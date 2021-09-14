import {Component} from '@angular/core';
import {MatDialogRef} from "@angular/material/dialog";
import {FormControl, FormGroup} from "@angular/forms";

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
  form = new FormGroup({
    numStars: new FormControl(3),
  });

  constructor(public readonly dialogRef: MatDialogRef<NewGameDialogComponent>) {
  }

  onNoClick(): void {
    this.dialogRef.close();
  }

}

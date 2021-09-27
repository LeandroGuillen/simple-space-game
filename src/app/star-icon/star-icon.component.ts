import { Component, Input } from '@angular/core';

@Component({
  selector: 'crn-star-icon',
  templateUrl: './star-icon.component.svg',
  styleUrls: [ './star-icon.component.scss' ]
})
export class StarIconComponent {
  @Input() color!: number;

  colors = [
    'white', // StarColorWhite
    'blue',// StarColorBlue
    'yellow', // StarColorYellow
    'orange', // StarColorOrange
    'red', // StarColorRed
  ]
}

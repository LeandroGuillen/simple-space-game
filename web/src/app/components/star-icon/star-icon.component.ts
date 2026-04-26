import { Component, Input } from '@angular/core';

@Component({
  selector: 'ssg-star-icon',
  standalone: false,
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

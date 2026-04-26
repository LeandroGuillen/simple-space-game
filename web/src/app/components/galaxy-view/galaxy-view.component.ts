import { Component, Input, OnInit } from '@angular/core';
import { Router } from '@angular/router';

import { Galaxy, Star } from '../../model/model';

@Component({
  selector: 'ssg-galaxy-view',
  standalone: false,
  templateUrl: './galaxy-view.component.svg',
  styleUrls: [ './galaxy-view.component.scss' ]
})
export class GalaxyViewComponent implements OnInit {
  @Input() galaxy!: Galaxy;

  colors = [
    'white', // StarColorWhite
    'blue',// StarColorBlue
    'yellow', // StarColorYellow
    'orange', // StarColorOrange
    'red', // StarColorRed
  ]

  constructor(private readonly router: Router) {
  }

  ngOnInit(): void {
  }

  clicked(star: Star) {
    this.router.navigate([ `stars/${star.id}` ])
  }

}

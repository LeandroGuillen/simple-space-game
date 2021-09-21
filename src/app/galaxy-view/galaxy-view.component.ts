import {Component, Input, OnInit} from '@angular/core';
import {Galaxy, Star} from "../model/model";
import {Router} from "@angular/router";

@Component({
  selector: 'crn-galaxy-view',
  templateUrl: './galaxy-view.component.svg',
  styleUrls: ['./galaxy-view.component.scss']
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
    this.router.navigate([`stars/${star.id}`])
  }

}

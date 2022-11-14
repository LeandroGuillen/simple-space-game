import { Component, Input, OnInit } from '@angular/core';

import { Building } from '../../model/model';

@Component({
  selector: 'ssg-building-icon',
  templateUrl: './building-icon.component.html',
  styleUrls: ['./building-icon.component.scss'],
})
export class BuildingIconComponent implements OnInit {
  @Input() building!: Building;

  constructor() {}

  ngOnInit(): void {}
}

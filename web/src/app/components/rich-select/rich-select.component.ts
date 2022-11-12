import { Component, Input, OnInit } from '@angular/core';

import { Planet } from '../../model/model';
import { StringService } from '../../services/string.service';

@Component({
  selector: 'ssg-rich-select',
  templateUrl: './rich-select.component.html',
  styleUrls: [ './rich-select.component.scss' ]
})
export class RichSelectComponent implements OnInit {
  shortDescription = '';

  @Input() planet!: Planet;
  @Input() selected: boolean = false;

  constructor(private readonly strings: StringService) { }

  ngOnInit(): void {
    this.shortDescription = this.strings.getPlanetShortDescription(this.planet);
  }

}

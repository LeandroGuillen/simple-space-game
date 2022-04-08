import { Component, OnInit } from '@angular/core';
import { FormBuilder, FormGroup } from '@angular/forms';

@Component({
  selector: 'ssg-ship-design',
  templateUrl: './ship-design.component.html',
  styleUrls: ['./ship-design.component.scss'],
})
export class ShipDesignComponent implements OnInit {
  form!: FormGroup;

  hullSizes = [
    'Frigate',
    'Destroyer',
    'Cruiser',
    'Battleship',
    'Titan',
    'Doom Star',
  ];
  constructor(readonly fb: FormBuilder) {
    this.form = fb.group({
      name: '',
      hullsize: '',
    });
  }

  ngOnInit(): void {}
}

import { Component, OnInit } from '@angular/core';
import { FormBuilder, FormGroup } from '@angular/forms';
import { MatTableDataSource } from '@angular/material/table';

@Component({
  selector: 'ssg-ship-design',
  templateUrl: './ship-design.component.html',
  styleUrls: ['./ship-design.component.scss'],
})
export class ShipDesignComponent implements OnInit {
  form!: FormGroup;

  weapons = []; // TODO max 8 elements
  weaponsTableDataSource: any; //MatTableDataSource;
  weaponsTableColumns = [
    { key: 'id', name: 'ID' },
    { key: 'amount', name: 'Amount' },
    { key: 'weapon', name: 'Weapon Type' },
    { key: 'damage', name: 'Damage' },
    { key: 'arc', name: 'Arc' },
    { key: 'cost', name: 'Cost' },
    { key: 'space', name: 'Space' },
    { key: 'mods', name: 'Modifications' },
  ];
  weaponsTableDisplayedColumns = this.weaponsTableColumns.map((c) => c.key);

  systems = []; // TODO max 8 elements
  systemsTableDataSource: any; //MatTableDataSource;
  systemsTableColumns = [
    { key: 'id', name: 'ID' },
    { key: 'system', name: 'Special System' },
    { key: 'description', name: 'Description' },
  ];
  systemsTableDisplayedColumns = this.systemsTableColumns.map((c) => c.key);

  hullSizes = [
    'Frigate',
    'Destroyer',
    'Cruiser',
    'Battleship',
    'Titan',
    'Doom Star',
  ];

  drives = [
    'Nuclear Drive',
    'Fusion Drive',
    'Ion Drive',
    'Hyper Drive',
    'Interphased Drive',
  ];

  shields = [
    'No Shield',
    'Class I Shield',
    'Class III Shield',
    'Class V Shield',
    'Class VII Shield',
    'Class X Shield',
  ];

  computers = [
    'Electronic Computer',
    'Optronic Computer',
    'Positronic Computer',
    'Cybertronic Computer',
    'Moleculatronic Computer',
  ];

  armors = ['Titanium', 'Tritanium', 'Zortrium', 'Neutronium', 'Adamantium'];

  constructor(readonly fb: FormBuilder) {
    this.form = fb.group({
      name: 'Nova',
      hullsize: 'Frigate',
      drive: 'Nuclear Drive',
      armor: 'Titanium',
      shield: 'No Shield',
      computer: 'Electronic Computer',
    });
  }

  ngOnInit(): void {
    this.weaponsTableDataSource = new MatTableDataSource(this.weapons);
    this.systemsTableDataSource = new MatTableDataSource(this.systems);
  }
}

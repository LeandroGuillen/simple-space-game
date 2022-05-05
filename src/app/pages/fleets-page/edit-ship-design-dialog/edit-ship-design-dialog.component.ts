import { Component, Inject, OnInit } from '@angular/core';
import { FormBuilder, FormGroup } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { MatTableDataSource } from '@angular/material/table';
import { Store } from '@ngrx/store';
import { Observable } from 'rxjs';
import { Armor, Computer, Drive, HullSize, Shield, ShipDesign } from 'src/app/model/combat';
import { AppState } from 'src/app/store/reducers';
import {
  selectArmors,
  selectComputers,
  selectDrives,
  selectHullSizes,
  selectShield,
} from 'src/app/store/selectors/game.selectors';

type EditShipDesignDialogData = {
  design: ShipDesign;
};

@Component({
  selector: 'ssg-edit-ship-design-dialog',
  templateUrl: './edit-ship-design-dialog.component.html',
  styleUrls: ['./edit-ship-design-dialog.component.scss'],
})
export class EditShipDesignDialogComponent implements OnInit {
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
  weaponsTableDisplayedColumns = this.weaponsTableColumns.map(c => c.key);

  systems = []; // TODO max 8 elements
  systemsTableDataSource: any; //MatTableDataSource;
  systemsTableColumns = [
    { key: 'id', name: 'ID' },
    { key: 'system', name: 'Special System' },
    { key: 'description', name: 'Description' },
  ];
  systemsTableDisplayedColumns = this.systemsTableColumns.map(c => c.key);

  hullSizes!: Observable<HullSize[]>;
  drives!: Observable<Drive[]>;
  shields!: Observable<Shield[]>;
  computers!: Observable<Computer[]>;
  armors!: Observable<Armor[]>;

  constructor(
    readonly fb: FormBuilder,
    public dialogRef: MatDialogRef<EditShipDesignDialogData>,
    @Inject(MAT_DIALOG_DATA) public data: EditShipDesignDialogData,
    private readonly store: Store<AppState>
  ) {
    this.form = fb.group({
      name: '',
      hullsize: '',
      drive: '',
      armor: '',
      shield: '',
      computer: '',
    });

    if (data) {
      this.form.patchValue({
        name: data.design.name,
        hullsize: data.design.hullSize.name,
        drive: data.design.drive.name,
        armor: data.design.armor.name,
        shield: data.design.shield.name,
        computer: data.design.computer.name,
      });
    }
  }

  ngOnInit(): void {
    this.weaponsTableDataSource = new MatTableDataSource(this.weapons);
    this.systemsTableDataSource = new MatTableDataSource(this.systems);
    this.hullSizes = this.store.select(selectHullSizes);
    this.drives = this.store.select(selectDrives);
    this.shields = this.store.select(selectShield);
    this.computers = this.store.select(selectComputers);
    this.armors = this.store.select(selectArmors);
  }

  cancel(): void {
    this.dialogRef.close();
  }
}

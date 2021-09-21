import {AfterViewInit, Component, OnInit, ViewChild} from '@angular/core';
import {MatTableDataSource} from "@angular/material/table";
import {MatSort} from "@angular/material/sort";
import {ColoniesPageService} from "./colonies-page.service";
import {Planet} from "../model/model";

export interface PeriodicElement {
  name: string;
  position: number;
  weight: number;
  symbol: string;
}

const ELEMENT_DATA: PeriodicElement[] = [
  {position: 1, name: 'Hydrogen', weight: 1.0079, symbol: 'H'},
  {position: 2, name: 'Helium', weight: 4.0026, symbol: 'He'},
  {position: 3, name: 'Lithium', weight: 6.941, symbol: 'Li'},
  {position: 4, name: 'Beryllium', weight: 9.0122, symbol: 'Be'},
  {position: 5, name: 'Boron', weight: 10.811, symbol: 'B'},
  {position: 6, name: 'Carbon', weight: 12.0107, symbol: 'C'},
  {position: 7, name: 'Nitrogen', weight: 14.0067, symbol: 'N'},
  {position: 8, name: 'Oxygen', weight: 15.9994, symbol: 'O'},
  {position: 9, name: 'Fluorine', weight: 18.9984, symbol: 'F'},
  {position: 10, name: 'Neon', weight: 20.1797, symbol: 'Ne'},
];

const planetData: Planet[] = []

@Component({
  selector: 'crn-colonies-page',
  templateUrl: './colonies-page.component.html',
  styleUrls: ['./colonies-page.component.scss']
})
export class ColoniesPageComponent implements AfterViewInit, OnInit {
  displayedColumns: (keyof Planet)[] = ['id', 'name', 'farmers', 'workers', 'scientists', 'current_project'];
  dataSource = new MatTableDataSource(planetData);

  colonyList!: Planet[];

  @ViewChild(MatSort) sort!: MatSort;

  constructor(private readonly coloniesPageService: ColoniesPageService) {
  }

  ngOnInit() {
    this.coloniesPageService.getColonyList().subscribe(colonies => {
      this.colonyList = colonies;
      this.dataSource = new MatTableDataSource(colonies);
      this.dataSource.sort = this.sort;
    })
  }

  ngAfterViewInit() {
    this.dataSource.sort = this.sort;
  }

}

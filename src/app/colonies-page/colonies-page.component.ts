import {AfterViewInit, Component, OnInit, ViewChild} from '@angular/core';
import {MatTableDataSource} from "@angular/material/table";
import {MatSort} from "@angular/material/sort";
import {ColoniesPageService} from "./colonies-page.service";
import {Planet} from "../model/model";
import {catchError, switchMap, tap} from "rxjs/operators";
import {of} from "rxjs";
import {MatDialog} from "@angular/material/dialog";
import {ChangeBuildingDialogComponent} from "./change-building-dialog/change-building-dialog.component";

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

  constructor(private readonly coloniesPageService: ColoniesPageService,
              private readonly dialog: MatDialog,) {
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

  changeBuilding(planet: Planet) {
    const oldProject = planet.current_project;
    this.dialog.open(ChangeBuildingDialogComponent).afterClosed().pipe(
      tap(buildingId => {
        planet.current_project = buildingId
      }),
      switchMap(buildingId => this.coloniesPageService.changeBuilding(planet.id, buildingId)),
      catchError(err => {
        // undo local change
        planet.current_project = oldProject
        console.log('error:', err)
        return of(false)
      })
    ).
    subscribe(response => {
      console.log(response)
    });
  }
}

import {Component, OnInit} from '@angular/core';
import {GameService} from "../game.service";
import {Galaxy} from "../model/model";
import {MatDialog} from "@angular/material/dialog";
import {catchError} from "rxjs/operators";
import {of} from "rxjs";
import {MatSnackBar} from "@angular/material/snack-bar";

@Component({
  selector: 'crn-galaxy-list',
  templateUrl: './star-list-page.component.html',
  styleUrls: ['./star-list-page.component.scss']
})
export class StarListPageComponent implements OnInit {
  galaxy!: Galaxy;

  constructor(private readonly gameService: GameService,
              private readonly dialog: MatDialog,
              private readonly snackBar: MatSnackBar) {
  }

  ngOnInit() {
    this.gameService.currentGalaxyUpdates.pipe(
      catchError(response => {
        console.error(response.error.message)
        this.gameService.clearCurrentGame()
        return of(null);
      }),
    ).subscribe(g => {
      if (g) {
        this.galaxy = g;
      } else {
        this.snackBar.open('cannot find current game', '', {duration: 5000});
      }
    });
  }
}

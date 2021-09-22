import {Component, OnInit} from '@angular/core';
import {MatDialog} from "@angular/material/dialog";
import {MatSnackBar} from "@angular/material/snack-bar";
import {GameListResponseItem, GamePageService} from "./game-page.service";
import {Router} from "@angular/router";
import {NewGameDialogComponent} from "./new-game-dialog/new-game-dialog.component";
import {catchError, switchMap} from "rxjs/operators";
import {of} from "rxjs";
import {GameService} from "../game.service";

@Component({
  selector: 'crn-game-page',
  templateUrl: './game-page.component.html',
  styleUrls: ['./game-page.component.scss']
})
export class GamePageComponent implements OnInit {
  gameList: GameListResponseItem[] = [];

  constructor(private readonly gamePageService: GamePageService,
              private readonly gameService: GameService,
              private readonly dialog: MatDialog,
              private readonly snackBar: MatSnackBar,
              private readonly route: Router) {
  }

  ngOnInit() {
    this.gamePageService.getGameList().subscribe(gameList => {
      this.gameList = gameList
    });
  }

  openNewGameDialog(): void {
    this.dialog.open(NewGameDialogComponent).afterClosed().pipe(
      switchMap(newGameRequest => this.gamePageService.newGame(newGameRequest)),
      switchMap(gameID => this.gameService.getCurrentGalaxy(gameID)),
      switchMap(() => this.gamePageService.getGameList()),
      catchError(err => {
        console.log('error:', err)
        return of(undefined)
      })
    ).subscribe(gameList => {
      if (gameList) {
        this.gameList = gameList;
      }
    });
  }

  startGame(gameID: string) {
    this.gameService.setCurrentGame(gameID);
    this.route.navigate(['stars']);
  }

}

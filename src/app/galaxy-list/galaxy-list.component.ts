import {Component, OnInit} from '@angular/core';
import {GameService} from "../game.service";
import {Galaxy} from "../model/model";
import {FormControl, FormGroup} from "@angular/forms";
import {MatDialog} from "@angular/material/dialog";
import {NewGameDialogComponent} from "./new-game-dialog/new-game-dialog.component";

@Component({
  selector: 'crn-galaxy-list',
  templateUrl: './galaxy-list.component.html',
  styleUrls: ['./galaxy-list.component.scss']
})
export class GalaxyListComponent implements OnInit {
  galaxy!: Galaxy;
  form = new FormGroup({
    numStars: new FormControl(3),
  });


  constructor(private readonly gameService: GameService, public readonly dialog: MatDialog) {
  }

  ngOnInit() {
    this.gameService.currentGalaxyUpdates.subscribe(g => this.galaxy = g);
  }


  openNewGameDialog(): void {
    this.dialog.open(NewGameDialogComponent).afterClosed().subscribe(numStars => {
      this.gameService.createGalaxy(numStars).subscribe()
    });
  }

}

import {Component, OnInit} from '@angular/core';
import {GameService} from "../game.service";
import {Galaxy} from "../model/model";
import {FormControl, FormGroup} from "@angular/forms";

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


  constructor(private readonly gameService: GameService) {
  }

  ngOnInit() {
    this.gameService.currentGalaxyUpdates.subscribe(g => this.galaxy = g);
  }

  newGame() {
    const numStars = this.form.value.numStars;
    this.gameService.createGalaxy(numStars).subscribe()
  }
}

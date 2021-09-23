import {Component, OnInit} from '@angular/core';
import {GameService} from "../game.service";
import {Observable} from "rxjs";
import {map} from "rxjs/operators";

@Component({
  selector: 'crn-top-bar',
  templateUrl: './top-bar.component.html',
  styleUrls: ['./top-bar.component.scss']
})
export class TopBarComponent implements OnInit {
  gameID!: Observable<string>;

  treasury!: number;

  constructor(private readonly gameService: GameService) {
  }

  ngOnInit(): void {
    this.gameID = this.gameService.currentGameID.pipe(map(id => id.substring(0, 8)));
  }

  turn() {
    this.gameService.turn().subscribe(res => {
      if (!res.ok) {
        return;
      }
      const player = res.game.players[0];
      this.treasury = player.treasury;
    });
  }
}

import { Injectable } from '@angular/core';
import { Action } from '@ngrx/store';
import { Observable } from 'rxjs';
import { map, shareReplay } from 'rxjs/operators';
import { webSocket } from 'rxjs/webSocket';

import { BEGame } from '../model/be-model';
import { mapGame } from '../model/mapper';
import { updateGame } from '../store/actions';

type GameUpdateMessage = {
  game_update: BEGame;
};

@Injectable({ providedIn: 'root' })
export class WebsocketService {
  private socket$!: Observable<Action>;

  getActions(): Observable<Action> {
    if (!this.socket$) {
      this.socket$ = webSocket<GameUpdateMessage>(
        'ws://localhost:8080/gameupdates'
      ).pipe(
        map(msg => msg.game_update),
        map(beGame => mapGame(beGame)),
        map(game => updateGame({ game })),
        shareReplay(1)
      );
    }
    return this.socket$;
  }
}

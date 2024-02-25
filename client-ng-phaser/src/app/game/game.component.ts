import { Component } from '@angular/core';
import { MainScene } from './scenes/main';
import { Game } from 'phaser';
import { GameScene } from './scenes/game';

@Component({
  selector: 'ssg-game',
  standalone: true,
  imports: [],
  templateUrl: './game.component.html',
  styleUrl: './game.component.scss',
})
export class GameComponent {
  phaserGame!: Game;
  config: Phaser.Types.Core.GameConfig;

  constructor() {
    this.config = {
      type: Phaser.AUTO,
      height: 600,
      width: 800,
      scene: [MainScene, GameScene],
      parent: 'gameContainer',
      title: 'Simple Space Game',
      backgroundColor: '#333',
      scale: {
        mode: Phaser.Scale.FIT,
        autoCenter: Phaser.Scale.CENTER_BOTH,
      },
      physics: {
        default: 'matter',
        matter: {
          debug: true,
          gravity: { x: 0, y: 0 },
        },
      },
    };
  }

  ngOnInit() {
    this.phaserGame = new Phaser.Game(this.config);
  }
}

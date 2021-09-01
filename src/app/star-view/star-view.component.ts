import {Component, OnInit} from '@angular/core';
import {Location} from '@angular/common';
import {GameService} from "../game.service";
import {ActivatedRoute} from "@angular/router";
import {map, switchMap} from "rxjs/operators";
import {Star} from "../model/model";

@Component({
  selector: 'crn-star-view',
  templateUrl: './star-view.component.html',
  styleUrls: ['./star-view.component.scss']
})
export class StarViewComponent implements OnInit {

  star!: Star;

  constructor(
    private readonly gameService: GameService,
    private readonly route: ActivatedRoute,
    private readonly location: Location) {
  }

  ngOnInit() {

    this.route.params.pipe(
      map(params => params['name']),
      switchMap(name => this.gameService.getStarByName(name))
    ).subscribe(star => {
      this.star = star;
    });
  }

  goBack() {
    this.location.back();
  }

}

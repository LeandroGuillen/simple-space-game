import { Component, OnInit } from '@angular/core';
import { AbstractControl, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { Router } from '@angular/router';

import { GameService } from '../../game.service';

type Server = {
  name: string;
  address: string;
}

@Component({
  selector: 'crn-home-page',
  templateUrl: './server-page.component.html',
  styleUrls: [ './server-page.component.scss' ]
})
export class ServerPageComponent implements OnInit {
  form!: FormGroup;

  rememberedServers: Server[] = [];

  constructor(
    private readonly fb: FormBuilder,
    private readonly gameService: GameService,
    private readonly route: Router) {
  }

  private static getRememberedServers(): Server[] {
    const servers = window.localStorage.getItem('rememberedServers');
    if (!servers) {
      return [];
    }

    return JSON.parse(servers);
  }

  private static setRememberedServers(servers: Server[]) {
    window.localStorage.setItem('rememberedServers', JSON.stringify(servers));
  }

  ngOnInit(): void {
    // initialize form
    this.form = this.fb.group({
      serverName: [ '', [ Validators.required, this.serverNameValidator.bind(this) ] ],
      serverAddress: [ '', Validators.required ],
    });

    // read local environment for possible servers
    this.rememberedServers = ServerPageComponent.getRememberedServers();
  }

  addServer() {
    this.form.markAsDirty();
    if (this.form.invalid) {
      return;
    }

    // add to servers to remember
    const server: Server = {
      name: this.form.value.serverName,
      address: this.form.value.serverAddress
    }

    this.addRememberedServer(server);

    this.form.updateValueAndValidity();
  }

  connect(server: Server) {
    // try to connect
    console.log('trying', server.address);

    this.gameService.host = server.address;
    this.gameService.establishConnection().subscribe(ok => {
      if (ok) {
        console.log('connection ok');
        this.route.navigate([ 'game' ]);
      } else {
        console.log('connection failed');
      }
    })
  }

  removeServer(s: Server) {
    const updatedServerList = this.rememberedServers.filter(server => server.name !== s.name);
    ServerPageComponent.setRememberedServers(updatedServerList);
    this.rememberedServers = ServerPageComponent.getRememberedServers();
  }

  private addRememberedServer(s: Server) {
    this.rememberedServers.unshift(s);
    ServerPageComponent.setRememberedServers(this.rememberedServers);
  }

  private serverNameValidator(control: AbstractControl) {
    const name = control.value;

    const exists = !!this.rememberedServers.find(s => s.name === name);
    if (exists) {
      return { duplicateServerName: true };
    }

    return null;
  }
}

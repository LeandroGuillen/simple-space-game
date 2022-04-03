import { Component, OnInit } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { Router } from '@angular/router';
import { Observable } from 'rxjs';
import { filter, tap } from 'rxjs/operators';

import { NewServerDialogComponent } from './new-server-dialog/new-server-dialog.component';
import { Server, ServerManagerService } from './server-manager.service';


@Component({
  selector: 'ssg-server-page',
  templateUrl: './server-page.component.html',
  styleUrls: [ './server-page.component.scss' ]
})
export class ServerPageComponent implements OnInit {
  servers!: Observable<Server[]>;
  numServers = 0;

  constructor(
    private readonly route: Router,
    private readonly dialog: MatDialog,
    private readonly serverManager: ServerManagerService
  ) { }

  ngOnInit(): void {
    this.servers = this.serverManager.getServerList().pipe(tap(servers => {
      this.numServers = servers.length;
    }));
  }

  openNewServerDialog(): void {
    this.dialog.open(NewServerDialogComponent).afterClosed()
      .pipe(filter(newServer => !!newServer))
      .subscribe(newServer => {
        this.serverManager.addServer(newServer.name, newServer.url);
      });
  }

  connect(server: Server) {
    this.serverManager.connectTo(server).subscribe(ok => {
      if (ok) {
        console.log('connection ok');
        this.route.navigate([ 'game' ]);
      } else {
        console.log('connection failed');
      }
    })
  }

  removeServer(server: Server) {
    this.serverManager.removeServer(server);
  }

}

import { Component, OnInit } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { Router } from '@angular/router';
import { Store } from '@ngrx/store';
import { Observable } from 'rxjs';
import { filter, tap } from 'rxjs/operators';

import { Server } from '../../model/model';
import { ServerManagerService } from '../../services/server-manager.service';
import { AppState } from '../../store/reducers';
import { selectServer } from '../../store/selectors/server.selectors';
import { NewServerDialogComponent } from './new-server-dialog/new-server-dialog.component';

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
    private readonly serverManager: ServerManagerService,
    private readonly store: Store<AppState>
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
    this.serverManager.connectTo(server);

    this.store.select(selectServer).subscribe(() => {
      this.route.navigate([ 'game' ]);
    });
  }

  removeServer(server: Server) {
    this.serverManager.removeServer(server);
  }

}

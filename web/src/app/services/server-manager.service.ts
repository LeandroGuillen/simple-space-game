import { Injectable } from '@angular/core';
import { Store } from '@ngrx/store';
import { BehaviorSubject, Observable } from 'rxjs';
import * as uuid from 'uuid';

import { Server } from '../model/model';
import { checkServerHealth, connectToServer } from '../store/actions';
import { AppState } from '../store/reducers';
import { selectServer } from '../store/selectors/server.selectors';

@Injectable({ providedIn: 'root' })
export class ServerManagerService {
  private serverListSubject = new BehaviorSubject<Server[]>([]);
  private readonly STORAGE_KEY = 'servers';
  private serverList: Server[] = [];

  constructor(private readonly store: Store<AppState>) {}

  connectTo(server: Server) {
    this.store.dispatch(connectToServer({ server }));
    return this.store.select(selectServer);
  }

  getServerList(): Observable<Server[]> {
    return this.serverListSubject.asObservable();
  }

  /** Read local environment for possible servers. */
  refreshServerList() {
    const serversString = window.localStorage.getItem(this.STORAGE_KEY);

    if (!serversString) {
      // no server found in storage
      this.serverListSubject.next([]);
    } else {
      // found some servers, check health
      const servers = JSON.parse(serversString as string) as Server[];
      servers.forEach(s => {
        this.store.dispatch(checkServerHealth({ server: s }));
      });
      this.serverList = servers;
      this.serverListSubject.next(servers);
    }
  }

  addServer(serverName: string, serverUrl: string) {
    const server: Server = {
      id: uuid.v4(),
      name: serverName,
      url: serverUrl,
      isOnline: false,
    };
    this.serverList.unshift(server);
    this.updateStorage(this.serverList);
    this.serverListSubject.next(this.serverList);
  }

  removeServer(s: Server) {
    const updatedServerList = this.serverList.filter(
      server => server.id !== s.id
    );
    this.updateStorage(updatedServerList);
    this.serverList = updatedServerList;
    this.serverListSubject.next(this.serverList);
  }

  private updateStorage(servers: Server[]) {
    window.localStorage.setItem(this.STORAGE_KEY, JSON.stringify(servers));
  }
}

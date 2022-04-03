import { Injectable } from '@angular/core';
import { Store } from '@ngrx/store';
import { BehaviorSubject, Observable } from 'rxjs';
import * as uuid from 'uuid';

import { Server } from '../model/model';
import { connectToServer } from '../store/actions';
import { AppState } from '../store/reducers';


@Injectable({ providedIn: "root" })
export class ServerManagerService {
	serverListSubject = new BehaviorSubject<Server[]>([]);
	serverList: Server[] = [];

	private readonly STORAGE_KEY = 'servers';

	constructor(private readonly store: Store<AppState>) { }

	connectTo(server: Server) {
		this.store.dispatch(connectToServer({ server }));
	}

	/** Read local environment for possible servers. */
	getServerList(): Observable<Server[]> {
		const serversString = window.localStorage.getItem(this.STORAGE_KEY);

		if (!serversString) {
			this.serverListSubject.next([]);
		} else {
			const servers = JSON.parse(serversString as string) as Server[];
			this.serverList = servers;
			this.serverListSubject.next(servers);
		}

		return this.serverListSubject.asObservable();
	}

	addServer(serverName: string, serverUrl: string) {
		const server: Server = {
			id: uuid.v4(),
			name: serverName,
			url: serverUrl
		}
		this.serverList.unshift(server);
		this.updateStorage(this.serverList);
		this.serverListSubject.next(this.serverList);
	}

	removeServer(s: Server) {
		const updatedServerList = this.serverList.filter(server => server.id !== s.id);
		this.updateStorage(updatedServerList);
		this.serverList = updatedServerList;
		this.serverListSubject.next(this.serverList);
	}

	private updateStorage(servers: Server[]) {
		window.localStorage.setItem(this.STORAGE_KEY, JSON.stringify(servers));
	}
}

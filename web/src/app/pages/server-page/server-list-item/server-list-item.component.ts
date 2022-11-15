import { Component, EventEmitter, Input, Output } from '@angular/core';

import { Server } from '../../../model/model';

@Component({
  selector: 'ssg-server-list-item',
  templateUrl: './server-list-item.component.html',
  styleUrls: ['./server-list-item.component.scss'],
})
export class ServerListItemComponent {
  @Input() server!: Server;
  @Input() status = false;

  @Output() connect = new EventEmitter<Server>();
  @Output() remove = new EventEmitter<Server>();
}

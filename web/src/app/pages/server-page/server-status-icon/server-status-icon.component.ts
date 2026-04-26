import { Component, Input, OnInit } from '@angular/core';

@Component({
  selector: 'ssg-server-status-icon',
  standalone: false,
  templateUrl: './server-status-icon.component.html',
  styleUrls: ['./server-status-icon.component.scss'],
})
export class ServerStatusIconComponent implements OnInit {
  @Input() status = false;
  constructor() {}

  ngOnInit(): void {}
}

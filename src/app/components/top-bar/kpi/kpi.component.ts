import { Component, Input } from '@angular/core';

@Component({
  selector: 'ssg-kpi',
  template: '<mat-icon>{{icon}}</mat-icon><span>{{value}}</span>',
  styleUrls: ['./kpi.component.scss'],
  host: { "[class.disabled]": "disabled" }
})
export class KpiComponent {

  @Input() icon!: string;
  @Input() value!: number | undefined | null;
  @Input() disabled = false;
}

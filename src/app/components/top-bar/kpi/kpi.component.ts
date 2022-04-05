import { Component, Input } from '@angular/core';

@Component({
  selector: 'ssg-kpi',
  templateUrl: './kpi.component.html',
  styleUrls: ['./kpi.component.scss'],
  host: { "[class.disabled]": "disabled" }
})
export class KpiComponent {
  @Input() icon!: string;
  @Input() value!: number | undefined | null;
  @Input() disabled = false;
}

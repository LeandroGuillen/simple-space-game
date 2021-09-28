import { Component, Input } from '@angular/core';

@Component({
  selector: 'crn-kpi',
  templateUrl: './kpi.component.html',
  styleUrls: [ './kpi.component.scss' ]
})
export class KpiComponent {

  @Input() icon!: string;
  @Input() value!: number | undefined | null;
}

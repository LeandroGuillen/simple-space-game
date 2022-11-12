import { Component, Input, OnInit } from '@angular/core';

@Component({
  selector: 'ssg-kpi',
  templateUrl: './kpi.component.html',
  styleUrls: ['./kpi.component.scss'],
  host: { "[class.disabled]": "disabled" }
})
export class KpiComponent implements OnInit{
  @Input() icon!: string;
  @Input() value!: number | undefined | null;
  @Input() disabled = false;
  @Input() decimals = 0;
  numberString: string = '1.0-0';

  ngOnInit() {
    this.numberString = `1.${this.decimals}-${ this.decimals }`
  }
}

import { Component, OnInit, ViewChild } from '@angular/core';
import { FormBuilder, FormGroup } from '@angular/forms';
import { MatSort } from '@angular/material/sort';
import { MatTableDataSource } from '@angular/material/table';

@Component({
  selector: 'ssg-fleet-table',
  templateUrl: './fleet-table.component.html',
  styleUrls: ['./fleet-table.component.scss'],
})
export class FleetTableComponent implements OnInit {
  filtersForm: FormGroup;
  displayedColumns = ['id', 'design'];
  dataSource: any; //MatTableDataSource;

  @ViewChild(MatSort) sort!: MatSort;

  constructor(readonly fb: FormBuilder) {
    this.filtersForm = fb.group({
      combat: { value: true, disabled: true },
      support: { value: true, disabled: true },
    });
  }

  ngOnInit(): void {
    this.dataSource = new MatTableDataSource([
      { id: 1, design: 'Creek I ' },
      { id: 2, design: 'Creek I ' },
      { id: 3, design: 'Creek VI ' },
      { id: 4, design: 'Creek VI ' },
      { id: 5, design: 'Creek IX ' },
      { id: 1, design: 'Creek I ' },
      { id: 2, design: 'Creek I ' },
      { id: 3, design: 'Creek VI ' },
      { id: 4, design: 'Creek VI ' },
      { id: 5, design: 'Creek IX ' },
      { id: 1, design: 'Creek I ' },
      { id: 2, design: 'Creek I ' },
      { id: 3, design: 'Creek VI ' },
      { id: 4, design: 'Creek VI ' },
      { id: 5, design: 'Creek IX ' },
      { id: 1, design: 'Creek I ' },
      { id: 2, design: 'Creek I ' },
      { id: 3, design: 'Creek VI ' },
      { id: 4, design: 'Creek VI ' },
      { id: 5, design: 'Creek IX ' },
    ]);
  }

  ngAfterViewInit() {
    this.dataSource.sort = this.sort;
  }
}

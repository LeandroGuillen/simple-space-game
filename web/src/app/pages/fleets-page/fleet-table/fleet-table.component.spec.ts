import { ComponentFixture, TestBed } from '@angular/core/testing';

import { FleetTableComponent } from './fleet-table.component';

describe('FleetTableComponent', () => {
  let component: FleetTableComponent;
  let fixture: ComponentFixture<FleetTableComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [ FleetTableComponent ]
    })
    .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(FleetTableComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

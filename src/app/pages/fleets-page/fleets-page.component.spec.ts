import { ComponentFixture, TestBed } from '@angular/core/testing';

import { FleetsPageComponent } from './fleets-page.component';

describe('FleetsPageComponent', () => {
  let component: FleetsPageComponent;
  let fixture: ComponentFixture<FleetsPageComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [ FleetsPageComponent ]
    })
    .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(FleetsPageComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

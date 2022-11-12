import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ShipDesignComponent } from './ship-design.component';

describe('ShipDesignComponent', () => {
  let component: ShipDesignComponent;
  let fixture: ComponentFixture<ShipDesignComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [ ShipDesignComponent ]
    })
    .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(ShipDesignComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

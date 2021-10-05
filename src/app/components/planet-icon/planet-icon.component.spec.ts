import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PlanetIconComponent } from './planet-icon.component';

describe('PlanetIconComponent', () => {
  let component: PlanetIconComponent;
  let fixture: ComponentFixture<PlanetIconComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [ PlanetIconComponent ]
    })
    .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(PlanetIconComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

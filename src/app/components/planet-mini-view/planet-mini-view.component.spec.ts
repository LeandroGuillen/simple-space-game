import { ComponentFixture, TestBed } from '@angular/core/testing';

import { PlanetMiniViewComponent } from './planet-mini-view.component';

describe('PlanetMiniViewComponent', () => {
  let component: PlanetMiniViewComponent;
  let fixture: ComponentFixture<PlanetMiniViewComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [ PlanetMiniViewComponent ]
    })
    .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(PlanetMiniViewComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

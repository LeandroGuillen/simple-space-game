import { ComponentFixture, TestBed } from '@angular/core/testing';

import { BuildingIconComponent } from './building-icon.component';

describe('BuildingIconComponent', () => {
  let component: BuildingIconComponent;
  let fixture: ComponentFixture<BuildingIconComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [ BuildingIconComponent ]
    })
    .compileComponents();

    fixture = TestBed.createComponent(BuildingIconComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

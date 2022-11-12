import { ComponentFixture, TestBed } from '@angular/core/testing';

import { GalaxyViewComponent } from './galaxy-view.component';

describe('GalaxyViewComponent', () => {
  let component: GalaxyViewComponent;
  let fixture: ComponentFixture<GalaxyViewComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [ GalaxyViewComponent ]
    })
    .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(GalaxyViewComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

import { ComponentFixture, TestBed } from '@angular/core/testing';

import { StarViewComponent } from './star-view.component';

describe('StarViewComponent', () => {
  let component: StarViewComponent;
  let fixture: ComponentFixture<StarViewComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [ StarViewComponent ]
    })
    .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(StarViewComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

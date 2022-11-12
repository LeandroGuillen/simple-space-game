import { ComponentFixture, TestBed } from '@angular/core/testing';

import { StarListPageComponent } from './star-list-page.component';

describe('StarListPageComponent', () => {
  let component: StarListPageComponent;
  let fixture: ComponentFixture<StarListPageComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [ StarListPageComponent ]
    })
    .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(StarListPageComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

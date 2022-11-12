import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ColoniesPageComponent } from './colonies-page.component';

describe('ColoniesPageComponent', () => {
  let component: ColoniesPageComponent;
  let fixture: ComponentFixture<ColoniesPageComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [ ColoniesPageComponent ]
    })
    .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(ColoniesPageComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

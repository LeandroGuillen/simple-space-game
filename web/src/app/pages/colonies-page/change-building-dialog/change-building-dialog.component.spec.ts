import {ComponentFixture, TestBed} from '@angular/core/testing';

import {ChangeBuildingDialogComponent} from './change-building-dialog.component';

describe('ChangeBuildingDialogComponent', () => {
  let component: ChangeBuildingDialogComponent;
  let fixture: ComponentFixture<ChangeBuildingDialogComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [ChangeBuildingDialogComponent]
    })
      .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(ChangeBuildingDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

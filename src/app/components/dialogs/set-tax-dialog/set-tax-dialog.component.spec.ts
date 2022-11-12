import { ComponentFixture, TestBed } from '@angular/core/testing';

import { SetTaxDialogComponent } from './set-tax-dialog.component';

describe('SetTaxDialogComponent', () => {
  let component: SetTaxDialogComponent;
  let fixture: ComponentFixture<SetTaxDialogComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [ SetTaxDialogComponent ]
    })
    .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(SetTaxDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

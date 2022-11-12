import { ComponentFixture, TestBed } from '@angular/core/testing';

import { NewServerDialogComponent } from './new-server-dialog.component';

describe('NewServerDialogComponent', () => {
  let component: NewServerDialogComponent;
  let fixture: ComponentFixture<NewServerDialogComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [ NewServerDialogComponent ]
    })
      .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(NewServerDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

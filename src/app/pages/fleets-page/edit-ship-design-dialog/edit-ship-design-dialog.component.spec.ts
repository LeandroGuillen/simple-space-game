import { ComponentFixture, TestBed } from '@angular/core/testing';

import { EditShipDesignDialogComponent } from './edit-ship-design-dialog.component';

describe('EditShipDesignDialogComponent', () => {
  let component: EditShipDesignDialogComponent;
  let fixture: ComponentFixture<EditShipDesignDialogComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [ EditShipDesignDialogComponent ]
    })
    .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(EditShipDesignDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

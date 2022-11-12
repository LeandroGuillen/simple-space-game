import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ServerStatusIconComponent } from './server-status-icon.component';

describe('ServerStatusIconComponent', () => {
  let component: ServerStatusIconComponent;
  let fixture: ComponentFixture<ServerStatusIconComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      declarations: [ ServerStatusIconComponent ]
    })
    .compileComponents();
  });

  beforeEach(() => {
    fixture = TestBed.createComponent(ServerStatusIconComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

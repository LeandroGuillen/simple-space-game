import { ComponentFixture, TestBed } from '@angular/core/testing';

import { ShipDesignTableComponent } from './ship-design-table.component';

describe('ShipDesignComponent', () => {
    let component: ShipDesignTableComponent;
    let fixture: ComponentFixture<ShipDesignTableComponent>;

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [ShipDesignTableComponent],
        }).compileComponents();
    });

    beforeEach(() => {
        fixture = TestBed.createComponent(ShipDesignTableComponent);
        component = fixture.componentInstance;
        fixture.detectChanges();
    });

    it('should create', () => {
        expect(component).toBeTruthy();
    });
});

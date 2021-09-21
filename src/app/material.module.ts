import {NgModule} from '@angular/core';
import {MatListModule} from '@angular/material/list';
import {MatTableModule} from '@angular/material/table';
import {MatButtonModule} from '@angular/material/button';
import {MatIconModule} from "@angular/material/icon";
import {MatInputModule} from "@angular/material/input";
import {MatDialogModule} from "@angular/material/dialog";
import {MatCheckboxModule} from "@angular/material/checkbox";
import {MatSnackBarModule} from "@angular/material/snack-bar";
import {MatToolbarModule} from "@angular/material/toolbar";
import {MatSliderModule} from "@angular/material/slider";
import {MatSlideToggleModule} from "@angular/material/slide-toggle";
import {MatSortModule} from "@angular/material/sort";

const moduleList = [
  MatListModule, MatTableModule, MatButtonModule, MatIconModule, MatInputModule, MatDialogModule, MatCheckboxModule, MatSnackBarModule, MatListModule, MatToolbarModule, MatSlideToggleModule, MatSortModule
]

@NgModule({
  imports: moduleList,
  exports: moduleList,
})
export class MaterialModule {
}

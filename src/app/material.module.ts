import {NgModule} from '@angular/core';
import {MatListModule} from '@angular/material/list';
import {MatTableModule} from '@angular/material/table';
import {MatButtonModule} from '@angular/material/button';
import {MatIconModule} from "@angular/material/icon";
import {MatFormFieldModule} from "@angular/material/form-field";
import {MatInputModule} from "@angular/material/input";

const moduleList = [
  MatListModule, MatTableModule, MatButtonModule, MatIconModule, MatInputModule
]

@NgModule({
  imports: moduleList,
  exports: moduleList,
})
export class MaterialModule {
}

import * as fromRouter from '@ngrx/router-store';
import { createSelector } from '@ngrx/store';

import { AppState } from '../reducers';

export const getRouterState = (state: AppState) => state.router;

export const getCurrentRouteState = createSelector(
	getRouterState,
	(router: fromRouter.RouterReducerState) => router.state
);



import { configureStore } from '@reduxjs/toolkit';
import selectionReducer from './selectionSlice';

export const makeStore = () =>
  configureStore({
    reducer: { selection: selectionReducer },
  });

export type AppStore = ReturnType<typeof makeStore>;
export type RootState = ReturnType<AppStore['getState']>;
export type AppDispatch = AppStore['dispatch'];
